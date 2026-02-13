package models

type HoursStatisticResponse struct {
	TotalHours    float32 `json:"totalHours"`
	StandardHours int32   `json:"standardHours"`
}

type WorkDaysStatisticResponse struct {
	TotalWorkDays    int64 `json:"totalWorkDays"`
	StandardWorkDays int32 `json:"standardWorkDays"`
}

type CountDaysResponse struct {
	Count int64 `json:"count"`
}
