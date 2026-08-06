package calendarevent

import (
	"context"
	"database/sql"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/date"
)

type service struct {
	repo *repo.Queries
}

type Service interface {
	GetCalendarEventsForMonth(ctx context.Context, month, year int) ([]repo.GetCalendarEventsForMonthRow, error)
	GetCalendarEventsForYear(ctx context.Context, year int) ([]repo.GetCalendarEventsForYearRow, error)
	GetCalendarEventByID(ctx context.Context, id string) (repo.GetCalendarEventsByIdRow, error)
	CreateCalendarEvent(ctx context.Context, eventDate time.Time, dayTypeID string, description string) error
	UpdateCalendarEvent(ctx context.Context, id string, eventDate time.Time, dayTypeID string, description string) error
	DeleteCalendarEvent(ctx context.Context, id string) error
}

func NewService(r *repo.Queries) Service {
	return &service{repo: r}
}

func (s *service) GetCalendarEventsForMonth(ctx context.Context, month, year int) ([]repo.GetCalendarEventsForMonthRow, error) {
	d := date.FirstDayOfMonth(month, year)
	return s.repo.GetCalendarEventsForMonth(ctx, repo.GetCalendarEventsForMonthParams{
		Year:  d,
		Month: d,
	})
}

func (s *service) GetCalendarEventsForYear(ctx context.Context, year int) ([]repo.GetCalendarEventsForYearRow, error) {
	return s.repo.GetCalendarEventsForYear(ctx, date.FirstDayOfMonth(1, year))
}

func (s *service) GetCalendarEventByID(ctx context.Context, id string) (repo.GetCalendarEventsByIdRow, error) {
	return s.repo.GetCalendarEventsById(ctx, id)
}

func (s *service) CreateCalendarEvent(ctx context.Context, eventDate time.Time, dayTypeID string, description string) error {
	var desc sql.NullString
	if description != "" {
		desc = sql.NullString{String: description, Valid: true}
	}
	_, err := s.repo.CreateCalendarEvents(ctx, repo.CreateCalendarEventsParams{
		EventDate:   eventDate,
		DayTypeID:   dayTypeID,
		Description: desc,
	})
	return err
}

func (s *service) UpdateCalendarEvent(ctx context.Context, id string, eventDate time.Time, dayTypeID string, description string) error {
	var desc sql.NullString
	if description != "" {
		desc = sql.NullString{String: description, Valid: true}
	}
	return s.repo.UpdateCalendarEvents(ctx, repo.UpdateCalendarEventsParams{
		ID:          id,
		EventDate:   eventDate,
		DayTypeID:   dayTypeID,
		Description: desc,
	})
}

func (s *service) DeleteCalendarEvent(ctx context.Context, id string) error {
	return s.repo.DeleteCalendarEvents(ctx, id)
}
