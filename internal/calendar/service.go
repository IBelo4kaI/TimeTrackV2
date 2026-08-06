package calendar

import (
	"context"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/date"

	"golang.org/x/sync/errgroup"
)

type service struct {
	repo repo.Querier
}

type Service interface {
	GetCalendarDays(ctx context.Context, userId string, month, year int) (*CalendarResponse, error)
}

func NewService(repo repo.Querier) Service {
	return &service{repo: repo}
}

func (s *service) GetCalendarDays(ctx context.Context, userId string, month, year int) (*CalendarResponse, error) {
	firstDayOfMonth := date.FirstDayOfMonth(month, year)
	lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, -1)
	daysInMonth := lastDayOfMonth.Day()

	g, ctx := errgroup.WithContext(ctx)

	var (
		timeEntries    []repo.UserTimeEntry
		calendarEvents []repo.GetCalendarEventsForMonthRow
		dayTypes       []repo.DayType
	)

	g.Go(func() error {
		var err error
		timeEntries, err = s.repo.GetUserTimeEntriesForMonth(ctx, repo.GetUserTimeEntriesForMonthParams{
			UserID: userId, Year: firstDayOfMonth, Month: firstDayOfMonth,
		})
		return err
	})

	g.Go(func() error {
		var err error
		calendarEvents, err = s.repo.GetCalendarEventsForMonth(ctx, repo.GetCalendarEventsForMonthParams{
			Year: firstDayOfMonth, Month: firstDayOfMonth,
		})
		return err
	})

	g.Go(func() error {
		var err error
		dayTypes, err = s.repo.GetDayTypes(ctx)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	userTimeEntriesMap := buildUserTimeEntriesMap(timeEntries)
	calendarEventsMap := buildCalendarEventsMap(calendarEvents)
	dayTypesMap := buildDayTypesMap(dayTypes)

	days := buildCalendarDays(year, month, daysInMonth, userTimeEntriesMap, calendarEventsMap, dayTypesMap)

	return &CalendarResponse{Days: days}, nil
}
