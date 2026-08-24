package sickleave

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/date"
	"timetrack/internal/notification"
	usertimeentry "timetrack/internal/user_time_entry"
	"timetrack/internal/vk"

	"github.com/google/uuid"
)

type Service interface {
	GetSickLeavesByYear(ctx context.Context, userID string, year int) ([]repo.GetSickLeavesByYearRow, error)
	GetAllUsersSickLeavesByYear(ctx context.Context, year int) ([]repo.GetAllUsersSickLeavesByYearRow, error)
	GetSickLeaveByID(ctx context.Context, id string) (repo.GetSickLeaveByIDRow, error)
	CreateSickLeave(ctx context.Context, p CreateSickLeaveParams) error
	UpdateSickLeaveStatus(ctx context.Context, id string, status repo.SickLeavesStatus) error
	DeleteSickLeave(ctx context.Context, id string) error
}

type CreateSickLeaveParams struct {
	UserID        string
	StartDate     time.Time
	EndDate       time.Time
	Description   string
	Status        repo.SickLeavesStatus
	ApplicantName string
}

type sickLeaveService struct {
	repo                 *repo.Queries
	userTimeEntryService usertimeentry.Service
	notificationService  notification.Service
	vkService            vk.Service
	frontendURL          string
}

func NewService(r *repo.Queries, userTimeEntryService usertimeentry.Service, notificationService notification.Service, vkService vk.Service, frontendURL string) Service {
	return &sickLeaveService{
		repo:                 r,
		userTimeEntryService: userTimeEntryService,
		notificationService:  notificationService,
		vkService:            vkService,
		frontendURL:          frontendURL,
	}
}

func (s *sickLeaveService) GetSickLeavesByYear(ctx context.Context, userID string, year int) ([]repo.GetSickLeavesByYearRow, error) {
	rows, err := s.repo.GetSickLeavesByYear(ctx, repo.GetSickLeavesByYearParams{
		UserID: userID,
		Year:   date.FirstDayOfMonth(1, year),
	})
	if err != nil {
		return nil, fmt.Errorf("get sick leaves: %w", err)
	}
	return rows, nil
}

func (s *sickLeaveService) GetAllUsersSickLeavesByYear(ctx context.Context, year int) ([]repo.GetAllUsersSickLeavesByYearRow, error) {
	rows, err := s.repo.GetAllUsersSickLeavesByYear(ctx, repo.GetAllUsersSickLeavesByYearParams{
		Year: date.FirstDayOfMonth(1, year),
	})
	if err != nil {
		return nil, fmt.Errorf("get all sick leaves: %w", err)
	}
	return rows, nil
}

func (s *sickLeaveService) GetSickLeaveByID(ctx context.Context, id string) (repo.GetSickLeaveByIDRow, error) {
	row, err := s.repo.GetSickLeaveByID(ctx, id)
	if err != nil {
		return repo.GetSickLeaveByIDRow{}, fmt.Errorf("get sick leave: %w", err)
	}
	return row, nil
}

func (s *sickLeaveService) CreateSickLeave(ctx context.Context, p CreateSickLeaveParams) error {
	totalDays := int32(p.EndDate.Sub(p.StartDate).Hours()/24) + 1

	var desc sql.NullString
	if p.Description != "" {
		desc = sql.NullString{String: p.Description, Valid: true}
	}

	status := p.Status
	if status == "" {
		status = repo.SickLeavesStatusUnofficial
	}

	id := uuid.NewString()

	if err := s.repo.CreateSickLeave(ctx, repo.CreateSickLeaveParams{
		ID:          id,
		UserID:      p.UserID,
		StartDate:   p.StartDate,
		EndDate:     p.EndDate,
		TotalDays:   totalDays,
		Description: desc,
		Status:      status,
	}); err != nil {
		return fmt.Errorf("create sick leave: %w", err)
	}

	if err := s.createSickLeaveTimeEntries(ctx, p.UserID, p.StartDate, p.EndDate); err != nil {
		return fmt.Errorf("create time entries: %w", err)
	}

	s.notifyAdminsNewApplication(ctx, id, p.StartDate, p.EndDate, p.ApplicantName)

	return nil
}

// notifyAdminsNewApplication — см. тот же метод у vacationService, здесь
// без ссылки на конкретную заявку: у больничных нет отдельной страницы-
// карточки (только общий список), ссылка ведёт на /sick-leave.
func (s *sickLeaveService) notifyAdminsNewApplication(ctx context.Context, id string, startDate, endDate time.Time, applicantName string) {
	adminIDs, err := s.notificationService.GetSickLeaveAdminRecipients(ctx)
	if err != nil || len(adminIDs) == 0 {
		return
	}

	title := "Новая заявка на больничный"
	body := fmt.Sprintf("%s – %s", startDate.Format("02.01.2006"), endDate.Format("02.01.2006"))
	if applicantName != "" {
		body = fmt.Sprintf("%s, %s", applicantName, body)
	}

	s.notificationService.CreateMany(ctx, adminIDs, title, body, repo.NotificationsTypeInfo, "sick_leave", id)
	s.vkService.NotifyMany(ctx, adminIDs, title+": "+body, s.frontendURL+"/sick-leave")
}

func (s *sickLeaveService) UpdateSickLeaveStatus(ctx context.Context, id string, status repo.SickLeavesStatus) error {
	return s.repo.UpdateSickLeaveStatus(ctx, repo.UpdateSickLeaveStatusParams{
		ID:     id,
		Status: status,
	})
}

func (s *sickLeaveService) DeleteSickLeave(ctx context.Context, id string) error {
	sl, err := s.repo.GetSickLeaveByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("sick leave not found")
		}
		return fmt.Errorf("get sick leave: %w", err)
	}

	if err := s.deleteSickLeaveTimeEntries(ctx, sl.UserID, sl.StartDate, sl.EndDate); err != nil {
		return fmt.Errorf("delete time entries: %w", err)
	}

	return s.repo.DeleteSickLeave(ctx, id)
}

// createSickLeaveTimeEntries создаёт user_time_entries с типом дня "medical" за период больничного.
func (s *sickLeaveService) createSickLeaveTimeEntries(ctx context.Context, userID string, startDate, endDate time.Time) error {
	medicalDayType, err := s.repo.GetDayTypeBySystemName(ctx, "medical")
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("get medical day type: %w", err)
	}

	existingMap, err := s.loadExistingEntries(ctx, userID, startDate, endDate)
	if err != nil {
		return err
	}

	var toCreate []repo.CreateUserTimeEntryParams
	var toUpdate []repo.UpdateUserTimeEntryParams

	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		day := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
		if existing, ok := existingMap[day]; ok {
			toUpdate = append(toUpdate, repo.UpdateUserTimeEntryParams{
				DayTypeID:   medicalDayType.ID,
				HoursWorked: existing.HoursWorked,
				EntryDate:   day,
				UserID:      userID,
			})
		} else {
			toCreate = append(toCreate, repo.CreateUserTimeEntryParams{
				UserID:      userID,
				EntryDate:   day,
				DayTypeID:   medicalDayType.ID,
				HoursWorked: "0",
			})
		}
	}

	if len(toCreate) > 0 {
		if err := s.userTimeEntryService.CreateUserTimeEntry(ctx, toCreate); err != nil {
			return fmt.Errorf("create entries: %w", err)
		}
	}
	if len(toUpdate) > 0 {
		if err := s.userTimeEntryService.UpdateUserTimeEntries(ctx, toUpdate); err != nil {
			return fmt.Errorf("update entries: %w", err)
		}
	}
	return nil
}

// deleteSickLeaveTimeEntries удаляет/восстанавливает user_time_entries с типом "medical".
func (s *sickLeaveService) deleteSickLeaveTimeEntries(ctx context.Context, userID string, startDate, endDate time.Time) error {
	medicalDayType, err := s.repo.GetDayTypeBySystemName(ctx, "medical")
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("get medical day type: %w", err)
	}

	workDayType, err := s.repo.GetDayTypeBySystemName(ctx, "work")
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("get work day type: %w", err)
	}

	existingMap, err := s.loadExistingEntries(ctx, userID, startDate, endDate)
	if err != nil {
		return err
	}

	var toDelete []time.Time
	var toUpdate []repo.UpdateUserTimeEntryParams

	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		day := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
		entry, ok := existingMap[day]
		if !ok || entry.DayTypeID != medicalDayType.ID {
			continue
		}

		hoursWorked := parseHours(entry.HoursWorked)
		if hoursWorked > 0 && workDayType.ID != "" {
			toUpdate = append(toUpdate, repo.UpdateUserTimeEntryParams{
				DayTypeID:   workDayType.ID,
				HoursWorked: entry.HoursWorked,
				EntryDate:   day,
				UserID:      userID,
			})
		} else {
			toDelete = append(toDelete, day)
		}
	}

	if len(toUpdate) > 0 {
		if err := s.userTimeEntryService.UpdateUserTimeEntries(ctx, toUpdate); err != nil {
			return fmt.Errorf("update entries: %w", err)
		}
	}
	if len(toDelete) > 0 {
		if err := s.userTimeEntryService.DeleteUserTimeEntries(ctx, repo.DeleteUserTimeEntriesParams{
			EntryDate: toDelete,
			UserID:    userID,
		}); err != nil {
			return fmt.Errorf("delete entries: %w", err)
		}
	}
	return nil
}

// loadExistingEntries загружает существующие записи за период в map[date]entry.
func (s *sickLeaveService) loadExistingEntries(ctx context.Context, userID string, startDate, endDate time.Time) (map[time.Time]repo.UserTimeEntry, error) {
	result := make(map[time.Time]repo.UserTimeEntry)

	monthSet := make(map[string]bool)
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		monthSet[fmt.Sprintf("%d-%02d", d.Year(), d.Month())] = true
	}

	for key := range monthSet {
		var year, month int
		fmt.Sscanf(key, "%d-%02d", &year, &month)
		firstDay := date.FirstDayOfMonth(month, year)
		entries, err := s.repo.GetUserTimeEntriesForMonth(ctx, repo.GetUserTimeEntriesForMonthParams{
			UserID: userID,
			Year:   firstDay,
			Month:  firstDay,
		})
		if err != nil {
			return nil, fmt.Errorf("get entries for %s: %w", key, err)
		}
		for _, e := range entries {
			day := time.Date(e.EntryDate.Year(), e.EntryDate.Month(), e.EntryDate.Day(), 0, 0, 0, 0, e.EntryDate.Location())
			result[day] = e
		}
	}
	return result, nil
}

func parseHours(s string) float64 {
	var h float64
	fmt.Sscanf(s, "%f", &h)
	return h
}
