package service

import (
	"context"
	"database/sql"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/date"
)

type calendarEventService struct {
	repo *repo.Queries
}

type CalendarEventService interface {
	GetCalendarEventsForMonth(ctx context.Context, month, year int) ([]repo.GetCalendarEventsForMonthRow, error)
	GetCalendarEventsForYear(ctx context.Context, year int) ([]repo.GetCalendarEventsForYearRow, error)
	GetCalendarEventByID(ctx context.Context, id string) (repo.GetCalendarEventsByIdRow, error)
	CreateCalendarEvent(ctx context.Context, eventDate time.Time, dayTypeID string, description string) error
	UpdateCalendarEvent(ctx context.Context, id string, eventDate time.Time, dayTypeID string, description string) error
	DeleteCalendarEvent(ctx context.Context, id string) error
}

func NewCalendarEventService(r *repo.Queries) CalendarEventService {
	return &calendarEventService{repo: r}
}

func (s *calendarEventService) GetCalendarEventsForMonth(ctx context.Context, month, year int) ([]repo.GetCalendarEventsForMonthRow, error) {
	d := date.FirstDayOfMonth(month, year)
	return s.repo.GetCalendarEventsForMonth(ctx, repo.GetCalendarEventsForMonthParams{
		Year:  d,
		Month: d,
	})
}

func (s *calendarEventService) GetCalendarEventsForYear(ctx context.Context, year int) ([]repo.GetCalendarEventsForYearRow, error) {
	return s.repo.GetCalendarEventsForYear(ctx, date.FirstDayOfMonth(1, year))
}

func (s *calendarEventService) GetCalendarEventByID(ctx context.Context, id string) (repo.GetCalendarEventsByIdRow, error) {
	return s.repo.GetCalendarEventsById(ctx, id)
}

func (s *calendarEventService) CreateCalendarEvent(ctx context.Context, eventDate time.Time, dayTypeID string, description string) error {
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

func (s *calendarEventService) UpdateCalendarEvent(ctx context.Context, id string, eventDate time.Time, dayTypeID string, description string) error {
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

func (s *calendarEventService) DeleteCalendarEvent(ctx context.Context, id string) error {
	return s.repo.DeleteCalendarEvents(ctx, id)
}
