package service

import (
	"context"
	"database/sql"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/date"
	"timetrack/internal/models"
)

type userTimeEntryService struct {
	repo *repo.Queries
	db   *sql.DB
}

type UserTimeEntryService interface {
	CreateUserTimeEntry(ctx context.Context, entries []repo.CreateUserTimeEntryParams) error
	DeleteUserTimeEntries(ctx context.Context, prm repo.DeleteUserTimeEntriesParams) error
	UpdateUserTimeEntries(ctx context.Context, entries []repo.UpdateUserTimeEntryParams) error
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

func (s *userTimeEntryService) GetStatisticsHoursByMonth(ctx context.Context, userId string, month int, year int, gender int) (*models.HoursStatisticResponse, error) {
	firstDayOfMonth := date.FirstDayOfMonth(month, year)

	totalHours, err := s.repo.GetTotalHoursByMonth(ctx, repo.GetTotalHoursByMonthParams{UserID: userId, Year: firstDayOfMonth, Month: firstDayOfMonth})

	if err != nil {
		return nil, err
	}

	standard, err := s.repo.GetWorkStandardsByMonthAndGenderId(ctx, repo.GetWorkStandardsByMonthAndGenderIdParams{
		Month:  int32(month),
		Year:   int32(year),
		Gender: int32(gender),
	})

	if err != nil {
		return nil, err
	}

	return &models.HoursStatisticResponse{
		TotalHours:    totalHours.(float32),
		StandardHours: standard.StandardHours,
	}, nil
}

func (s *userTimeEntryService) GetStatisticsWorkDaysByMonth(ctx context.Context, userId string, month int, year int, gender int) (*models.WorkDaysStatisticResponse, error) {
	firstDayOfMonth := date.FirstDayOfMonth(month, year)

	totalDays, err := s.repo.GetWorkDaysByMonth(ctx, repo.GetWorkDaysByMonthParams{UserID: userId, Month: firstDayOfMonth, Year: firstDayOfMonth})

	if err != nil {
		return nil, err
	}

	standard, err := s.repo.GetWorkStandardsByMonthAndGenderId(ctx, repo.GetWorkStandardsByMonthAndGenderIdParams{
		Month:  int32(month),
		Year:   int32(year),
		Gender: int32(gender),
	})

	if err != nil {
		return nil, err
	}

	return &models.WorkDaysStatisticResponse{
		TotalWorkDays:    totalDays,
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
		Count: countDays,
	}, nil
}


