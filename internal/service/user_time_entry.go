package service

import (
	"context"
	"database/sql"
	repo "timetrack/internal/adapter/mysql/sqlc"
)

type userTimeEntryService struct {
	repo *repo.Queries
	db   *sql.DB
}

type UserTimeEntryService interface {
	CreateUserTimeEntry(ctx context.Context, entries []repo.CreateUserTimeEntryParams) error
	DeleteUserTimeEntries(ctx context.Context, ids []string) error
	UpdateUserTimeEntries(ctx context.Context, prm repo.UpdateUserTimeEntriesParams) error
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

func (s *userTimeEntryService) UpdateUserTimeEntries(ctx context.Context, prm repo.UpdateUserTimeEntriesParams) error {
	return s.repo.UpdateUserTimeEntries(ctx, prm)
}

func (s *userTimeEntryService) DeleteUserTimeEntries(ctx context.Context, ids []string) error {
	return s.repo.DeleteUserTimeEntries(ctx, ids)
}
