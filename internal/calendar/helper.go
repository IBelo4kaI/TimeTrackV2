package calendar

import (
	"strconv"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
)

func buildUserTimeEntriesMap(entries []repo.UserTimeEntry) map[time.Time]repo.UserTimeEntry {
	m := make(map[time.Time]repo.UserTimeEntry, len(entries))
	for _, entry := range entries {
		d := normalizeDate(entry.EntryDate)
		m[d] = entry
	}
	return m
}

func buildCalendarEventsMap(events []repo.GetCalendarEventsForMonthRow) map[time.Time][]repo.GetCalendarEventsForMonthRow {
	m := make(map[time.Time][]repo.GetCalendarEventsForMonthRow)
	for _, event := range events {
		d := normalizeDate(event.EventDate)
		m[d] = append(m[d], event)
	}
	return m
}

func buildDayTypesMap(dayTypes []repo.DayType) map[string]repo.DayType {
	m := make(map[string]repo.DayType, len(dayTypes))
	for _, dt := range dayTypes {
		m[dt.ID] = dt
	}
	return m
}

func buildCalendarDays(
	year, month, daysInMonth int,
	userTimeEntriesMap map[time.Time]repo.UserTimeEntry,
	calendarEventsMap map[time.Time][]repo.GetCalendarEventsForMonthRow,
	dayTypesMap map[string]repo.DayType,
) []CalendarDay {
	days := make([]CalendarDay, 0, daysInMonth)

	for day := 1; day <= daysInMonth; day++ {
		currentDate := normalizeDate(time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC))

		calendarDay := CalendarDay{
			Date:       currentDate,
			Hours:      0,
			Holidays:   []string{},
			IsWeekend:  currentDate.Weekday() == time.Saturday || currentDate.Weekday() == time.Sunday,
			IsEditType: true,
		}

		if entry, exists := userTimeEntriesMap[currentDate]; exists {
			applyUserTimeEntry(&calendarDay, entry, dayTypesMap)
		}

		if events, exists := calendarEventsMap[currentDate]; exists {
			applyCalendarEvents(&calendarDay, events, dayTypesMap)
		}

		days = append(days, calendarDay)
	}

	return days
}

func normalizeDate(t time.Time) time.Time {
	return time.Date(
		t.Year(),
		t.Month(),
		t.Day(),
		0, 0, 0, 0,
		time.UTC,
	)
}

func applyUserTimeEntry(day *CalendarDay, entry repo.UserTimeEntry, dayTypesMap map[string]repo.DayType) {
	if hours, err := strconv.ParseFloat(entry.HoursWorked, 32); err == nil {
		day.Hours = float32(hours)
	}
	day.UserTimeId = entry.ID
	day.UserTimeTypeId = entry.DayTypeID
	day.IsEditType = dayTypesMap[entry.DayTypeID].SystemName != "vacation"
}

func applyCalendarEvents(day *CalendarDay, events []repo.GetCalendarEventsForMonthRow, dayTypesMap map[string]repo.DayType) {
	for _, event := range events {
		day.CalendarEventTypeId = event.DayTypeID
		day.IsWeekend = !dayTypesMap[event.DayTypeID].IsWorkDay

		if event.Description != "" {
			day.Holidays = append(day.Holidays, event.Description)
		}
	}
}
