package service

import (
	"context"
	"fmt"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/date"
	"timetrack/internal/models"
)

type calendarService struct {
	repo repo.Querier
}

type CalendarService interface {
	GetCalendarDays(ctx context.Context, userId string, month, year int) (*models.CalendarResponse, error)
}

func NewCalendarService(repo repo.Querier) CalendarService {
	return &calendarService{repo: repo}
}

func (s *calendarService) GetCalendarDays(ctx context.Context, userId string, month, year int) (*models.CalendarResponse, error) {
	// Определяем количество дней в месяце
	firstDayOfMonth := date.FirstDayOfMonth(month, year)
	lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, -1)
	daysInMonth := lastDayOfMonth.Day()

	userTimeEntries, err := s.repo.GetUserTimeEntriesForMonth(ctx, repo.GetUserTimeEntriesForMonthParams{
		UserID: userId, Year: firstDayOfMonth, Month: firstDayOfMonth,
	})
	if err != nil {
		return nil, err
	}

	// Получаем календарные события за месяц
	calendarEvents, err := s.repo.GetCalendarEventsForMonth(ctx, repo.GetCalendarEventsForMonthParams{
		Year:  firstDayOfMonth,
		Month: firstDayOfMonth,
	})
	if err != nil {
		return nil, err
	}

	dayTypes, err := s.repo.GetDayTypes(ctx)
	if err != nil {
		return nil, err
	}

	// Создаем мапы для быстрого поиска
	userTimeEntriesMap := make(map[time.Time]repo.UserTimeEntry)
	for _, entry := range userTimeEntries {
		entryDate := date.NewDate(entry.EntryDate.Day(), int(entry.EntryDate.Month()), entry.EntryDate.Year())
		userTimeEntriesMap[entryDate] = entry
	}

	// Изменяем тип map - теперь храним слайс событий
	calendarEventsMap := make(map[time.Time][]repo.GetCalendarEventsForMonthRow)

	for _, event := range calendarEvents {
		eventDate := date.NewDate(event.EventDate.Day(), int(event.EventDate.Month()), event.EventDate.Year())
		// Добавляем событие в слайс для этой даты
		calendarEventsMap[eventDate] = append(calendarEventsMap[eventDate], event)
	}

	dayTypesMap := make(map[string]repo.DayType)
	for _, dayType := range dayTypes {
		dayTypesMap[dayType.ID] = dayType
	}

	// Создаем массив дней
	days := make([]models.CalendarDay, 0, daysInMonth)

	for day := 1; day <= daysInMonth; day++ {
		currentDate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

		// Создаем базовый объект дня
		calendarDay := models.CalendarDay{
			Date:       currentDate,
			Hours:      0,
			Holidays:   []string{},
			IsWeekend:  (currentDate.Weekday() == time.Saturday || currentDate.Weekday() == time.Sunday),
			IsEditType: true,
		}
		// Заполняем данные из userTimeEntries если есть
		if entry, exists := userTimeEntriesMap[currentDate]; exists {
			// Преобразуем hoursWorked из string в float32
			var hours float32
			_, err := fmt.Sscanf(entry.HoursWorked, "%f", &hours)
			if err == nil {
				calendarDay.Hours = hours
			}
			calendarDay.UserTimeId = entry.ID
			calendarDay.UserTimeTypeId = entry.DayTypeID
			//TODO: В настройках сервиса создать "можно ли редактировать день с типом отпуска" сейчас нельзя.
			calendarDay.IsEditType = dayTypesMap[entry.DayTypeID].SystemName != "vacation"
		}

		// Заполняем данные из calendarEvents если есть
		if events, exists := calendarEventsMap[currentDate]; exists {
			// Обрабатываем все события для этой даты
			for _, event := range events {
				// Устанавливаем тип дня (берем последний или приоритетный)
				calendarDay.CalendarEventTypeId = event.DayTypeID
				calendarDay.IsWeekend = !dayTypesMap[event.DayTypeID].IsWorkDay

				// Добавляем описание в список праздников
				if event.Description != "" {
					calendarDay.Holidays = append(calendarDay.Holidays, event.Description)
				}
			}
		}

		days = append(days, calendarDay)
	}

	return &models.CalendarResponse{
		Days: days,
	}, nil
}
