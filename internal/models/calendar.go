package models

import "time"

type CalendarResponse struct {
	Days []CalendarDay `json:"days"`
}
type CalendarDay struct {
	Date                time.Time `json:"date"`
	Hours               float32   `json:"hours"`
	UserTimeId          string    `json:"userTimeId"`
	UserTimeTypeId      string    `json:"userTimeTypeId"`
	CalendarEventTypeId string    `json:"calendarEventTypeId"`
	Holidays            []string  `json:"holidays"`
	IsWeekend           bool      `json:"isWeekend"`
	IsEditType          bool      `json:"isEditType"`
}
