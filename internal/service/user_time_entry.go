package service

import (
	"context"
	"database/sql"
	"strconv"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/date"
	"timetrack/internal/models"
	"timetrack/internal/parser"
)

type userTimeEntryService struct {
	repo *repo.Queries
	db   *sql.DB
}

type UserTimeEntryService interface {
	CreateUserTimeEntry(ctx context.Context, entries []repo.CreateUserTimeEntryParams) error
	DeleteUserTimeEntries(ctx context.Context, prm repo.DeleteUserTimeEntriesParams) error
	UpdateUserTimeEntries(ctx context.Context, entries []repo.UpdateUserTimeEntryParams) error
	GetStatisticsHoursByMonth(ctx context.Context, userId string, month int, year int, gender int) (*models.HoursStatisticResponse, error)
	GetStatisticsWorkDaysByMonth(ctx context.Context, userId string, month int, year int, gender int) (*models.WorkDaysStatisticResponse, error)
	GetCountDaysByMonthWithSystemName(ctx context.Context, userId string, month int, year int, gender int, systemName string) (*models.CountDaysResponse, error)
	GetVacationStatistics(ctx context.Context, userId string, year int) (*models.VacationStatisticsResponse, error)
}

func NewUserTimeEntryService(repo *repo.Queries, db *sql.DB) UserTimeEntryService {
	return &userTimeEntryService{repo: repo, db: db}
}

func (s *userTimeEntryService) CreateUserTimeEntry(ctx context.Context, entries []repo.CreateUserTimeEntryParams) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)

	for _, entry := range entries {
		err = qtx.CreateUserTimeEntry(ctx, entry)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *userTimeEntryService) UpdateUserTimeEntries(ctx context.Context, entries []repo.UpdateUserTimeEntryParams) error {

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)

	for _, entry := range entries {
		err = qtx.UpdateUserTimeEntry(ctx, entry)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *userTimeEntryService) DeleteUserTimeEntries(ctx context.Context, prm repo.DeleteUserTimeEntriesParams) error {
	return s.repo.DeleteUserTimeEntries(ctx, prm)
}

// getWorkStandardWithPriority получает норму работы с приоритетом индивидуальных данных
// Сначала пытается получить индивидуальную норму для пользователя, если не найдена - общую норму
func (s *userTimeEntryService) getWorkStandardWithPriority(ctx context.Context, userId string, month, year, gender int32) (repo.WorkStandard, error) {
	// Сначала пытаемся получить индивидуальную норму для пользователя
	standard, err := s.repo.GetWorkStandardsByMonthAndGenderIdAndUserId(ctx, repo.GetWorkStandardsByMonthAndGenderIdAndUserIdParams{
		Month:  month,
		Year:   year,
		Gender: gender,
		UserID: sql.NullString{
			String: userId,
			Valid:  true,
		},
	})

	// Если индивидуальная норма не найдена, получаем общую норму
	if err != nil && err == sql.ErrNoRows {
		standard, err = s.repo.GetWorkStandardsByMonthAndGenderId(ctx, repo.GetWorkStandardsByMonthAndGenderIdParams{
			Month:  month,
			Year:   year,
			Gender: gender,
		})
	}

	return standard, err
}

func (s *userTimeEntryService) GetStatisticsHoursByMonth(ctx context.Context, userId string, month int, year int, gender int) (*models.HoursStatisticResponse, error) {
	firstDayOfMonth := date.FirstDayOfMonth(month, year)

	totalHours, err := s.repo.GetTotalHoursByMonth(ctx, repo.GetTotalHoursByMonthParams{UserID: userId, Year: firstDayOfMonth, Month: firstDayOfMonth})

	if err != nil {
		return nil, err
	}

	standard, err := s.getWorkStandardWithPriority(ctx, userId, int32(month), int32(year), int32(gender))
	if err != nil {
		if err == sql.ErrNoRows {
			return &models.HoursStatisticResponse{
				TotalHours:    parser.InterfaceToFloat32(totalHours),
				StandardHours: 0,
			}, nil
		}
		return nil, err
	}

	return &models.HoursStatisticResponse{
		TotalHours:    parser.InterfaceToFloat32(totalHours),
		StandardHours: standard.StandardHours,
	}, nil
}

func (s *userTimeEntryService) GetStatisticsWorkDaysByMonth(ctx context.Context, userId string, month int, year int, gender int) (*models.WorkDaysStatisticResponse, error) {
	firstDayOfMonth := date.FirstDayOfMonth(month, year)

	totalDays, err := s.repo.GetWorkDaysByMonth(ctx, repo.GetWorkDaysByMonthParams{UserID: userId, Month: firstDayOfMonth, Year: firstDayOfMonth})

	if err != nil {
		return nil, err
	}

	standard, err := s.getWorkStandardWithPriority(ctx, userId, int32(month), int32(year), int32(gender))
	if err != nil {
		if err == sql.ErrNoRows {
			return &models.WorkDaysStatisticResponse{
				TotalWorkDays:    parser.InterfaceToInt64(totalDays),
				StandardWorkDays: 0,
			}, nil
		}
		return nil, err
	}

	return &models.WorkDaysStatisticResponse{
		TotalWorkDays:    parser.InterfaceToInt64(totalDays),
		StandardWorkDays: standard.StandardDays,
	}, nil
}

func (s *userTimeEntryService) GetCountDaysByMonthWithSystemName(ctx context.Context, userId string, month int, year int, gender int, systemName string) (*models.CountDaysResponse, error) {
	firstDayOfMonth := date.FirstDayOfMonth(month, year)

	countDays, err := s.repo.GetTotalDaysByMonthWithSystemName(ctx, repo.GetTotalDaysByMonthWithSystemNameParams{
		UserID:     userId,
		Year:       firstDayOfMonth,
		Month:      firstDayOfMonth,
		SystemName: systemName,
	})

	if err != nil {
		return nil, err
	}

	return &models.CountDaysResponse{
		Count: parser.InterfaceToInt64(countDays),
	}, nil
}

func (s *userTimeEntryService) GetVacationStatistics(ctx context.Context, userId string, year int) (*models.VacationStatisticsResponse, error) {
	// Получаем использованные дни отпуска за год
	firstDayOfYear := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	usedVacationDaysInterface, err := s.repo.GetVacationDaysByYear(ctx, repo.GetVacationDaysByYearParams{
		UserID: userId,
		Year:   firstDayOfYear,
	})

	if err != nil {
		return nil, err
	}

	usedVacationDays := parser.InterfaceToInt64(usedVacationDaysInterface)

	// Получаем общее количество отпускных дней из настроек системы
	setting, err := s.repo.GetSystemSettingByKey(ctx, "vacation_duration")
	if err != nil {
		// Если настройка не найдена, используем значение по умолчанию
		if err == sql.ErrNoRows {
			return &models.VacationStatisticsResponse{
				UsedVacationDays:      usedVacationDays,
				TotalVacationDays:     30, // Стандартное значение по ТК РФ
				RemainingVacationDays: 30 - usedVacationDays,
			}, nil
		}
		return nil, err
	}

	// Преобразуем значение из строки в число
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

	// Рассчитываем оставшиеся дни
	remainingVacationDays := max(totalVacationDays-usedVacationDays, 0)

	return &models.VacationStatisticsResponse{
		UsedVacationDays:      usedVacationDays,
		TotalVacationDays:     totalVacationDays,
		RemainingVacationDays: remainingVacationDays,
	}, nil
}
