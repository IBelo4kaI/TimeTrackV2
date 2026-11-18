package timesheet

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/timeaudit"
)

type Service struct{ repo *repo.Queries }

func NewService(queries *repo.Queries) *Service { return &Service{repo: queries} }
func monthStart(userID string, year, month int) (time.Time, error) {
	if userID == "" || year < 1 || year > 9999 || month < 1 || month > 12 {
		return time.Time{}, fmt.Errorf("invalid timesheet owner or month")
	}
	return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC), nil
}
func (s *Service) GetMonth(ctx context.Context, userID string, year, month int) (repo.TimesheetMonth, error) {
	if _, err := monthStart(userID, year, month); err != nil {
		return repo.TimesheetMonth{}, err
	}
	state, err := s.repo.GetTimesheetMonth(ctx, repo.GetTimesheetMonthParams{UserID: userID, Year: int16(year), Month: int8(month)})
	if errors.Is(err, sql.ErrNoRows) {
		return repo.TimesheetMonth{UserID: userID, Year: int16(year), Month: int8(month), Status: repo.TimesheetMonthsStatusPending}, nil
	}
	return state, err
}
func (s *Service) History(ctx context.Context, userID string, year, month int) ([]repo.TimeEntryHistory, error) {
	start, err := monthStart(userID, year, month)
	if err != nil {
		return nil, err
	}
	result, err := s.repo.ListTimeEntryHistory(ctx, repo.ListTimeEntryHistoryParams{UserID: userID, FromDate: start, ToDate: start.AddDate(0, 1, 0)})
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = []repo.TimeEntryHistory{}
	}
	return result, nil
}

// Decide locks the same month row used by every audited time writer.
func (s *Service) Decide(ctx context.Context, db *sql.DB, userID string, year, month int, expected int64, approve bool) (repo.TimesheetMonth, error) {
	start, err := monthStart(userID, year, month)
	if err != nil || expected < 0 {
		return repo.TimesheetMonth{}, fiber.ErrBadRequest
	}
	actor := timeaudit.FromContext(ctx)
	if err := timeaudit.Validate(ctx); err != nil {
		return repo.TimesheetMonth{}, fiber.ErrUnauthorized
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return repo.TimesheetMonth{}, err
	}
	defer tx.Rollback()
	q := s.repo.WithTx(tx)
	if err := q.LockTimeEntryMonths(ctx, []repo.TimeEntryKey{{UserID: userID, Date: start}}); err != nil {
		return repo.TimesheetMonth{}, err
	}
	key := repo.LockTimesheetMonthParams{UserID: userID, Year: int16(year), Month: int8(month)}
	state, err := q.LockTimesheetMonth(ctx, key)
	if err != nil {
		return state, err
	}
	if state.Revision != expected {
		return state, fiber.NewError(409, "Табель изменился: загрузите актуальные данные")
	}
	if approve {
		if state.Status != repo.TimesheetMonthsStatusApproved {
			err = q.ApproveTimesheetMonth(ctx, repo.ApproveTimesheetMonthParams{ApprovedBy: sql.NullString{String: actor.ID, Valid: true}, UserID: userID, Year: int16(year), Month: int8(month)})
		}
	} else {
		err = q.UnapproveTimesheetMonth(ctx, repo.UnapproveTimesheetMonthParams{UserID: userID, Year: int16(year), Month: int8(month)})
	}
	if err != nil {
		return state, err
	}
	state, err = q.LockTimesheetMonth(ctx, key)
	if err != nil {
		return state, err
	}
	return state, tx.Commit()
}
