package models

type HoursStatisticResponse struct {
	TotalHours    any   `json:"totalHours"`
	StandardHours int32 `json:"standardHours"`
}

type WorkDaysStatisticResponse struct {
	TotalWorkDays    int64 `json:"totalWorkDays"`
	StandardWorkDays int32 `json:"standardWorkDays"`
}

type CountDaysResponse struct {
	Count int64 `json:"count"`
}

type VacationStatisticsResponse struct {
	UsedVacationDays      int64 `json:"usedVacationDays"`
	TotalVacationDays     int64 `json:"totalVacationDays"`
	RemainingVacationDays int64 `json:"remainingVacationDays"`
}

type ReportStatisticsResponse struct {
	Hours        HoursStatisticResponse    `json:"hours"`
	WorkDays     WorkDaysStatisticResponse `json:"workDays"`
	VacationDays CountDaysResponse         `json:"vacationDays"`
	MedicalDays  CountDaysResponse         `json:"medicalDays"`
	TimeOffDays  CountDaysResponse         `json:"timeoffDays"`
	DecreeDays   CountDaysResponse         `json:"decreeDays"`
}
