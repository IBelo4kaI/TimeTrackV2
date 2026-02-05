package service

import (
	"context"
	repo "timetrack/internal/adapter/mysql/sqlc"
)

type userTimeEntryService struct {
	repo repo.Querier
}

type UserTimeEntryService interface {
	DeleteUserTimeEntries(ctx context.Context, ids []string) error
	UpdateUserTimeEntries(ctx context.Context, ids []string, dayTypeId string, hoursWorked string) error
}

func NewUserTimeEntryService(repo repo.Querier) UserTimeEntryService {
	return &userTimeEntryService{repo: repo}
}

func (s *userTimeEntryService) UpdateUserTimeEntries(ctx context.Context, ids []string, dayTypeId string, hoursWorked string) error {
	return s.repo.UpdateUserTimeEntries(ctx, repo.UpdateUserTimeEntriesParams{
		Ids:         ids,
		DayTypeID:   dayTypeId,
		HoursWorked: hoursWorked,
	})
}

func (s *userTimeEntryService) DeleteUserTimeEntries(ctx context.Context, ids []string) error {
	return s.repo.DeleteUserTimeEntries(ctx, ids)
}
