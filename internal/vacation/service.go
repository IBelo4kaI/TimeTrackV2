package vacation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/date"
	"timetrack/internal/notification"
	"timetrack/internal/parser"
	usertimeentry "timetrack/internal/user_time_entry"
	"timetrack/internal/vk"

	"github.com/google/uuid"
)

type vacationService struct {
	repo                 *repo.Queries
	db                   *sql.DB
	userTimeEntryService usertimeentry.Service
	notificationService  notification.Service
	vkService            vk.Service
	frontendURL          string
}

type Service interface {
	CalculateVacationDaysFullMonths(ctx context.Context, startDate, endDate time.Time) (*VacationCalculationResult, error)
	CalculateVacationDays(ctx context.Context, startDate, endDate time.Time) (*VacationCalculationResult, error)
	GetCountVacationsByStatus(ctx context.Context, prm repo.GetCountVacationsByStatusParams) (int, error)
	GetVacationsStats(ctx context.Context, userId string, year int) (*VacationStats, error)
	GetAllUserVacationsByYear(ctx context.Context, year int) (*[]repo.GetAllUsersVacationsByYearRow, error)
	// ListVacationCalendarByYear — то же самое, но без description, для
	// виджета "отпуска коллег" (см. ту же ручку/разрешение
	// time:vacation_calendar:read в route.go — специально отдельно от
	// vacation.all:read).
	ListVacationCalendarByYear(ctx context.Context, year int) (*[]repo.ListVacationCalendarByYearRow, error)
	GetVacationsByYear(ctx context.Context, userId string, year int) (*[]repo.GetVacationsByYearRow, error)
	GetVacationByID(ctx context.Context, vacationID string) (*repo.GetVacationByIDRow, error)
	CreateVacationReport(ctx context.Context, vacation VacationCreateRequest) error
	ApproveVacation(ctx context.Context, vacationID string) error
	UpdateVacationStatus(ctx context.Context, vacationID string, newStatus repo.VacationsStatus) error
	UpdateVacationType(ctx context.Context, vacationID string, vacationTypeID string) error
	DeleteVacation(ctx context.Context, vacationID string) error
}

// defaultVacationTypeSystemName — тип отпуска, назначаемый по умолчанию, если клиент его не указал.
const defaultVacationTypeSystemName = "paid"

func NewService(repo *repo.Queries, db *sql.DB, userTimeEntryService usertimeentry.Service, notificationService notification.Service, vkService vk.Service, frontendURL string) Service {
	return &vacationService{
		repo:                 repo,
		db:                   db,
		userTimeEntryService: userTimeEntryService,
		notificationService:  notificationService,
		vkService:            vkService,
		frontendURL:          frontendURL,
	}
}

func (s *vacationService) GetVacationsByYear(ctx context.Context, userId string, year int) (*[]repo.GetVacationsByYearRow, error) {
	vacations, err := s.repo.GetVacationsByYear(ctx, repo.GetVacationsByYearParams{UserID: userId, Year: date.FirstDayOfMonth(1, year)})
	if err != nil {
		return nil, err
	}

	return &vacations, nil
}

func (s *vacationService) GetAllUserVacationsByYear(ctx context.Context, year int) (*[]repo.GetAllUsersVacationsByYearRow, error) {
	vacations, err := s.repo.GetAllUsersVacationsByYear(ctx, repo.GetAllUsersVacationsByYearParams{Year: date.FirstDayOfMonth(1, year)})
	if err != nil {
		return nil, err
	}

	return &vacations, nil
}

func (s *vacationService) ListVacationCalendarByYear(ctx context.Context, year int) (*[]repo.ListVacationCalendarByYearRow, error) {
	vacations, err := s.repo.ListVacationCalendarByYear(ctx, repo.ListVacationCalendarByYearParams{Year: date.FirstDayOfMonth(1, year)})
	if err != nil {
		return nil, err
	}

	return &vacations, nil
}

func (s *vacationService) GetVacationByID(ctx context.Context, vacationID string) (*repo.GetVacationByIDRow, error) {
	vacation, err := s.repo.GetVacationByID(ctx, vacationID)
	if err != nil {
		return nil, err
	}

	return &vacation, nil
}

func (s *vacationService) CreateVacationReport(ctx context.Context, vacation VacationCreateRequest) error {
	totalDays, err := s.CalculateVacationDays(ctx, vacation.StartDate, vacation.EndDate)
	if err != nil {
		return err
	}

	var desc sql.NullString

	if vacation.Description != "" {
		desc = sql.NullString{String: vacation.Description, Valid: true}
	} else {
		desc = sql.NullString{Valid: false}
	}

	vacationTypeID, err := s.resolveVacationTypeID(ctx, vacation.VacationTypeID)
	if err != nil {
		return err
	}

	vacationID := uuid.NewString()

	// Создаем отпуск в базе данных
	err = s.repo.CreateVacation(ctx, repo.CreateVacationParams{
		ID:             vacationID,
		UserID:         vacation.UserID,
		StartDate:      vacation.StartDate,
		EndDate:        vacation.EndDate,
		TotalDays:      int32(totalDays.TotalVacationDays),
		Description:    desc,
		Status:         vacation.Status,
		VacationTypeID: vacationTypeID,
	})
	if err != nil {
		return err
	}

	// Если статус отпуска "approved", создаем userTimeEntry
	if vacation.Status == repo.VacationsStatusApproved {
		err = s.createOrUpdateVacationTimeEntries(ctx, vacation.UserID, vacation.StartDate, vacation.EndDate)
		if err != nil {
			return fmt.Errorf("failed to create vacation time entries: %w", err)
		}
	}

	dates := fmt.Sprintf("%s – %s", vacation.StartDate.Format("02.01.2006"), vacation.EndDate.Format("02.01.2006"))
	body := dates
	if vacation.ApplicantName != "" {
		body = fmt.Sprintf("%s, %s", vacation.ApplicantName, dates)
	}

	s.notifyAdminsNewApplication(ctx, "vacation", vacationID, "Новая заявка на отпуск", body)

	return nil
}

// notifyAdminsNewApplication — и в notifications (таблица), и в VK, тем, кто
// настроен в system_settings под отпуска отдельно от больничных (см.
// notification.Service.GetVacationAdminRecipients). Best-effort — ошибка
// тут не должна ронять создание самой заявки.
func (s *vacationService) notifyAdminsNewApplication(ctx context.Context, entityType, entityID, title, body string) {
	adminIDs, err := s.notificationService.GetVacationAdminRecipients(ctx)
	if err != nil || len(adminIDs) == 0 {
		return
	}

	s.notificationService.CreateMany(ctx, adminIDs, title, body, repo.NotificationsTypeInfo, entityType, entityID)
	s.vkService.NotifyMany(ctx, adminIDs, title+": "+body, fmt.Sprintf("%s/docs/vacation/%s", s.frontendURL, entityID))
}

// Основная версия — только дни между датами, сгруппированные по месяцам
func (s *vacationService) CalculateVacationDays(ctx context.Context, startDate, endDate time.Time) (*VacationCalculationResult, error) {
	if startDate.After(endDate) {
		startDate, endDate = endDate, startDate
	}
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location())

	var result VacationCalculationResult
	monthMap := make(map[string]*VacationMonth) // ключ: "YYYY-MM"
	var monthOrder []string

	for currentDate := startDate; !currentDate.After(endDate); currentDate = currentDate.AddDate(0, 0, 1) {
		day, isVacation := s.buildVacationDay(ctx, currentDate)
		if isVacation {
			result.TotalVacationDays++
		}

		key := currentDate.Format("2006-01")
		if _, exists := monthMap[key]; !exists {
			monthMap[key] = &VacationMonth{
				Year:  currentDate.Year(),
				Month: currentDate.Month(),
			}
			monthOrder = append(monthOrder, key)
		}
		monthMap[key].Days = append(monthMap[key].Days, day)
	}

	for _, key := range monthOrder {
		result.Months = append(result.Months, *monthMap[key])
	}

	return &result, nil
}

// Версия с полными месяцами — каждый месяц раскрывается целиком
func (s *vacationService) CalculateVacationDaysFullMonths(ctx context.Context, startDate, endDate time.Time) (*VacationCalculationResult, error) {
	if startDate.After(endDate) {
		startDate, endDate = endDate, startDate
	}

	// Начало первого месяца и конец последнего
	fullStart := time.Date(startDate.Year(), startDate.Month(), 1, 0, 0, 0, 0, startDate.Location())
	fullEnd := time.Date(endDate.Year(), endDate.Month()+1, 0, 0, 0, 0, 0, endDate.Location()) // последний день месяца

	// Нормализуем границы отпуска для подсчёта TotalVacationDays
	vacationStart := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	vacationEnd := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location())

	var result VacationCalculationResult
	monthMap := make(map[string]*VacationMonth)
	var monthOrder []string

	for currentDate := fullStart; !currentDate.After(fullEnd); currentDate = currentDate.AddDate(0, 0, 1) {
		inVacationRange := !currentDate.Before(vacationStart) && !currentDate.After(vacationEnd)

		day, isVacation := s.buildVacationDay(ctx, currentDate)

		if !inVacationRange {
			day.IsVacation = false
			isVacation = false
		}

		if isVacation && inVacationRange {
			result.TotalVacationDays++
		}

		key := currentDate.Format("2006-01")
		if _, exists := monthMap[key]; !exists {
			monthMap[key] = &VacationMonth{
				Year:  currentDate.Year(),
				Month: currentDate.Month(),
			}
			monthOrder = append(monthOrder, key)
		}
		monthMap[key].Days = append(monthMap[key].Days, day)
	}

	for _, key := range monthOrder {
		result.Months = append(result.Months, *monthMap[key])
	}

	return &result, nil
}

func (s *vacationService) GetCountVacationsByStatus(ctx context.Context, prm repo.GetCountVacationsByStatusParams) (int, error) {

	result, err := s.repo.GetCountVacationsByStatus(ctx, prm)

	if err != nil {
		return 0, err
	}

	fmt.Printf("%+v\n", result)

	return parser.InterfaceToInt(result), nil
}

func (s *vacationService) GetVacationsStats(ctx context.Context, userId string, year int) (*VacationStats, error) {
	yearDate := date.NewDate(1, 1, year)

	approveds, err := s.repo.GetCountVacationsByStatus(ctx, repo.GetCountVacationsByStatusParams{
		UserID: userId,
		Status: repo.VacationsStatusApproved,
		Year:   yearDate,
	})

	if err != nil {
		return nil, err
	}

	pendings, err := s.repo.GetCountVacationsByStatus(ctx, repo.GetCountVacationsByStatusParams{
		UserID: userId,
		Status: repo.VacationsStatusPending,
		Year:   yearDate,
	})

	if err != nil {
		return nil, err
	}
	// Получаем общее количество отпускных дней из настроек системы
	setting, err := s.repo.GetSystemSettingByKey(ctx, "vacation_duration")
	if err != nil {
		// Если настройка не найдена, используем значение по умолчанию
		if err == sql.ErrNoRows {
			return &VacationStats{Used: 0, Pending: 0, Free: 0}, nil
		}
		return nil, err
	}

	var settingValue string
	if setting.SettingValue.Valid {
		settingValue = setting.SettingValue.String
	} else {
		// Если значение NULL, используем значение по умолчанию
		settingValue = "30"
	}

	totalVacationDays, err := strconv.ParseInt(settingValue, 10, 64)
	if err != nil {
		// Если не удалось преобразовать, используем значение по умолчанию
		totalVacationDays = 30
	}

	remainingVacationDays := max(totalVacationDays-parser.InterfaceToInt64(approveds), 0)

	return &VacationStats{
		Used:    parser.InterfaceToInt(approveds),
		Pending: parser.InterfaceToInt(pendings),
		Free:    int(remainingVacationDays),
	}, nil
}

// resolveVacationTypeID возвращает sql.NullString с ID типа отпуска. Если типа явно не
// передали, подставляется тип по умолчанию (system_name = "paid"); если такого типа
// в базе ещё нет (например, миграция не накатилась), отпуск создаётся без типа.
func (s *vacationService) resolveVacationTypeID(ctx context.Context, vacationTypeID string) (sql.NullString, error) {
	if vacationTypeID != "" {
		if _, err := s.repo.GetVacationTypeByID(ctx, vacationTypeID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return sql.NullString{}, fmt.Errorf("тип отпуска не найден")
			}
			return sql.NullString{}, err
		}
		return sql.NullString{String: vacationTypeID, Valid: true}, nil
	}

	defaultType, err := s.repo.GetVacationTypeBySystemName(ctx, defaultVacationTypeSystemName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sql.NullString{}, nil
		}
		return sql.NullString{}, err
	}
	return sql.NullString{String: defaultType.ID, Valid: true}, nil
}

func (s *vacationService) UpdateVacationType(ctx context.Context, vacationID string, vacationTypeID string) error {
	typeID, err := s.resolveVacationTypeID(ctx, vacationTypeID)
	if err != nil {
		return err
	}
	if !typeID.Valid {
		return fmt.Errorf("тип отпуска не найден")
	}

	if err := s.repo.AssignVacationType(ctx, repo.AssignVacationTypeParams{
		VacationTypeID: typeID,
		ID:             vacationID,
	}); err != nil {
		return fmt.Errorf("failed to update vacation type: %w", err)
	}
	return nil
}

// Вспомогательный метод — логика определения типа дня (вынесена из оригинала)
func (s *vacationService) buildVacationDay(ctx context.Context, date time.Time) (VacationDay, bool) {
	calendarEvent, err := s.repo.GetCalendarEventsByDate(ctx, date)
	var isVacation bool
	var description string

	if err != nil {
		isVacation = true
	} else {
		dayType, err := s.repo.GetDayTypeByID(ctx, calendarEvent.DayTypeID)
		if err != nil {
			isVacation = true
			description = calendarEvent.Description
		} else {
			isVacation = dayType.AffectsVacation
			description = calendarEvent.Description
		}
	}

	return VacationDay{
		Date:        date,
		IsVacation:  isVacation,
		Description: description,
	}, isVacation
}

// Вспомогательный метод — создание или обновление userTimeEntry для отпускных дней
func (s *vacationService) createOrUpdateVacationTimeEntries(ctx context.Context, userID string, startDate, endDate time.Time) error {
	// Получаем day_type_id для типа "vacation"
	dayType, err := s.repo.GetDayTypeBySystemName(ctx, "vacation")
	if err != nil {
		// Если day_type не найден, просто возвращаем успех без создания записей
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("failed to get vacation day type: %w", err)
	}

	// Используем CalculateVacationDays для определения, какие дни являются отпускными
	vacationDays, err := s.CalculateVacationDays(ctx, startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to calculate vacation days: %w", err)
	}

	// Собираем все даты, которые являются отпускными (isVacation = true)
	vacationDates := make(map[time.Time]bool)
	for _, month := range vacationDays.Months {
		for _, day := range month.Days {
			if day.IsVacation {
				// Нормализуем дату (убираем время)
				date := time.Date(day.Date.Year(), day.Date.Month(), day.Date.Day(), 0, 0, 0, 0, day.Date.Location())
				vacationDates[date] = true
			}
		}
	}

	// Получаем все существующие записи за период отпуска
	existingEntriesMap := make(map[time.Time]repo.UserTimeEntry)

	// Собираем уникальные месяцы в периоде отпуска
	monthSet := make(map[string]bool)
	for date := range vacationDates {
		monthKey := fmt.Sprintf("%d-%02d", date.Year(), date.Month())
		monthSet[monthKey] = true
	}

	// Получаем записи для каждого месяца
	for monthKey := range monthSet {
		var year, month int
		_, err := fmt.Sscanf(monthKey, "%d-%02d", &year, &month)
		if err != nil {
			continue
		}

		firstDayOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, startDate.Location())
		existingEntries, err := s.repo.GetUserTimeEntriesForMonth(ctx, repo.GetUserTimeEntriesForMonthParams{
			UserID: userID,
			Year:   firstDayOfMonth,
			Month:  firstDayOfMonth,
		})
		if err != nil {
			return fmt.Errorf("failed to get user time entries for month %s: %w", monthKey, err)
		}

		// Добавляем записи в map для быстрого поиска
		for _, entry := range existingEntries {
			existingEntriesMap[entry.EntryDate] = entry
		}
	}

	// Создаем userTimeEntry для каждого отпускного дня
	var entries []repo.CreateUserTimeEntryParams
	var updateEntries []repo.UpdateUserTimeEntryParams

	// Проходим по всем отпускным дням
	for date := range vacationDates {
		// Проверяем, существует ли уже запись для этой даты
		if _, exists := existingEntriesMap[date]; exists {
			// Запись существует, добавляем в список для обновления
			// Сохраняем текущее значение часов, не сбрасываем на "0"
			updateEntries = append(updateEntries, repo.UpdateUserTimeEntryParams{
				DayTypeID:   dayType.ID,
				HoursWorked: existingEntriesMap[date].HoursWorked,
				EntryDate:   date,
				UserID:      userID,
			})
		} else {
			// Если записи нет, добавляем в список для создания
			entries = append(entries, repo.CreateUserTimeEntryParams{
				UserID:      userID,
				EntryDate:   date,
				DayTypeID:   dayType.ID,
				HoursWorked: "0",
			})
		}
	}

	// Создаем новые записи
	if len(entries) > 0 {
		err = s.userTimeEntryService.CreateUserTimeEntry(ctx, entries)
		if err != nil {
			return fmt.Errorf("failed to create user time entries: %w", err)
		}
	}

	// Обновляем существующие записи
	if len(updateEntries) > 0 {
		err = s.userTimeEntryService.UpdateUserTimeEntries(ctx, updateEntries)
		if err != nil {
			return fmt.Errorf("failed to update user time entries: %w", err)
		}
	}

	return nil
}

// Вспомогательный метод — удаление userTimeEntry для отпускных дней
func (s *vacationService) deleteVacationTimeEntries(ctx context.Context, userID string, startDate, endDate time.Time) error {
	// Используем CalculateVacationDays для определения, какие дни являются отпускными
	vacationDays, err := s.CalculateVacationDays(ctx, startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to calculate vacation days: %w", err)
	}

	// Собираем все даты, которые являются отпускными (isVacation = true)
	var vacationDates []time.Time
	for _, month := range vacationDays.Months {
		for _, day := range month.Days {
			if day.IsVacation {
				// Нормализуем дату (убираем время)
				date := time.Date(day.Date.Year(), day.Date.Month(), day.Date.Day(), 0, 0, 0, 0, day.Date.Location())
				vacationDates = append(vacationDates, date)
			}
		}
	}

	// Получаем day_type_id для типа "vacation"
	vacationDayType, err := s.repo.GetDayTypeBySystemName(ctx, "vacation")
	if err != nil {
		// Если day_type не найден, просто возвращаем успех без удаления записей
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("failed to get vacation day type: %w", err)
	}

	// Получаем day_type_id для типа "work"
	workDayType, err := s.repo.GetDayTypeBySystemName(ctx, "work")
	if err != nil {
		// Если day_type "work" не найден, просто удаляем записи
		if err == sql.ErrNoRows {
			workDayType = repo.DayType{}
		} else {
			return fmt.Errorf("failed to get work day type: %w", err)
		}
	}

	// Получаем все существующие записи за период отпуска
	existingEntriesMap := make(map[time.Time]repo.UserTimeEntry)

	// Собираем уникальные месяцы в периоде отпуска
	monthSet := make(map[string]bool)
	for _, date := range vacationDates {
		monthKey := fmt.Sprintf("%d-%02d", date.Year(), date.Month())
		monthSet[monthKey] = true
	}

	// Получаем записи для каждого месяца
	for monthKey := range monthSet {
		var year, month int
		_, err := fmt.Sscanf(monthKey, "%d-%02d", &year, &month)
		if err != nil {
			continue
		}

		firstDayOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, startDate.Location())
		existingEntries, err := s.repo.GetUserTimeEntriesForMonth(ctx, repo.GetUserTimeEntriesForMonthParams{
			UserID: userID,
			Year:   firstDayOfMonth,
			Month:  firstDayOfMonth,
		})
		if err != nil {
			return fmt.Errorf("failed to get user time entries for month %s: %w", monthKey, err)
		}

		// Добавляем записи в map для быстрого поиска
		for _, entry := range existingEntries {
			existingEntriesMap[entry.EntryDate] = entry
		}
	}

	// Обрабатываем записи с типом "vacation" для отпускных дней
	var entriesToDelete []time.Time
	var entriesToUpdate []repo.UpdateUserTimeEntryParams

	for _, date := range vacationDates {
		if entry, exists := existingEntriesMap[date]; exists {
			// Проверяем, что запись имеет тип "vacation"
			if entry.DayTypeID == vacationDayType.ID {
				// Проверяем количество часов
				hours, err := strconv.ParseFloat(entry.HoursWorked, 64)
				if err != nil {
					// Если не удалось преобразовать часы, считаем их равными 0
					hours = 0
				}

				if hours > 0 && workDayType.ID != "" {
					// Если часов больше 0 и есть тип "work", обновляем запись на тип "work"
					entriesToUpdate = append(entriesToUpdate, repo.UpdateUserTimeEntryParams{
						DayTypeID:   workDayType.ID,
						HoursWorked: entry.HoursWorked, // Сохраняем текущее значение часов
						EntryDate:   date,
						UserID:      userID,
					})
				} else {
					// Если часов 0 или нет типа "work", удаляем запись
					entriesToDelete = append(entriesToDelete, date)
				}
			}
		}
	}

	// Обновляем записи с часами > 0 на тип "work"
	if len(entriesToUpdate) > 0 {
		err = s.userTimeEntryService.UpdateUserTimeEntries(ctx, entriesToUpdate)
		if err != nil {
			return fmt.Errorf("failed to update user time entries to work type: %w", err)
		}
	}

	// Удаляем записи с часами = 0
	if len(entriesToDelete) > 0 {
		err = s.userTimeEntryService.DeleteUserTimeEntries(ctx, repo.DeleteUserTimeEntriesParams{
			EntryDate: entriesToDelete,
			UserID:    userID,
		})
		if err != nil {
			return fmt.Errorf("failed to delete user time entries: %w", err)
		}
	}

	return nil
}

func (s *vacationService) ApproveVacation(ctx context.Context, vacationID string) error {
	// Получаем отпуск по ID
	vacation, err := s.repo.GetVacationByID(ctx, vacationID)
	if err != nil {
		return fmt.Errorf("failed to get vacation: %w", err)
	}

	// Проверяем, что статус отпуска "pending"
	if vacation.Status != repo.VacationsStatusPending {
		return fmt.Errorf("vacation status is not pending, current status: %s", vacation.Status)
	}

	// Создаем или обновляем записи userTimeEntry для отпускных дней
	err = s.createOrUpdateVacationTimeEntries(ctx, vacation.UserID, vacation.StartDate, vacation.EndDate)
	if err != nil {
		return fmt.Errorf("failed to create vacation time entries: %w", err)
	}

	// Обновляем статус отпуска на "approved"
	err = s.repo.UpdateVacationStatus(ctx, repo.UpdateVacationStatusParams{
		ID:     vacationID,
		Status: repo.VacationsStatusApproved,
	})
	if err != nil {
		return fmt.Errorf("failed to update vacation status: %w", err)
	}

	return nil
}

func (s *vacationService) UpdateVacationStatus(ctx context.Context, vacationID string, newStatus repo.VacationsStatus) error {
	// Получаем отпуск по ID
	vacation, err := s.repo.GetVacationByID(ctx, vacationID)
	if err != nil {
		return fmt.Errorf("failed to get vacation: %w", err)
	}

	// Если новый статус "approved", вызываем метод ApproveVacation
	if newStatus == repo.VacationsStatusApproved {
		return s.ApproveVacation(ctx, vacationID)
	}

	// Если текущий статус "approved" и новый статус не "approved",
	// нужно удалить userTimeEntry для отпускных дней
	if vacation.Status == repo.VacationsStatusApproved && newStatus != repo.VacationsStatusApproved {
		err = s.deleteVacationTimeEntries(ctx, vacation.UserID, vacation.StartDate, vacation.EndDate)
		if err != nil {
			return fmt.Errorf("failed to delete vacation time entries: %w", err)
		}
	}

	// Обновляем статус отпуска
	err = s.repo.UpdateVacationStatus(ctx, repo.UpdateVacationStatusParams{
		ID:     vacationID,
		Status: newStatus,
	})
	if err != nil {
		return fmt.Errorf("failed to update vacation status: %w", err)
	}

	return nil
}

func (s *vacationService) DeleteVacation(ctx context.Context, vacationID string) error {
	// Получаем отпуск по ID, чтобы проверить его статус
	vacation, err := s.repo.GetVacationByID(ctx, vacationID)
	if err != nil {
		// Если отпуск не найден, возвращаем ошибку
		if err == sql.ErrNoRows {
			return fmt.Errorf("vacation not found")
		}
		return fmt.Errorf("failed to get vacation: %w", err)
	}

	// Если отпуск был утвержден, удаляем связанные userTimeEntry
	if vacation.Status == repo.VacationsStatusApproved {
		err = s.deleteVacationTimeEntries(ctx, vacation.UserID, vacation.StartDate, vacation.EndDate)
		if err != nil {
			return fmt.Errorf("failed to delete vacation time entries: %w", err)
		}
	}

	// Удаляем отпуск
	err = s.repo.DeleteVacation(ctx, vacationID)
	if err != nil {
		return fmt.Errorf("failed to delete vacation: %w", err)
	}

	return nil
}
