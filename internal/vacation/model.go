package vacation

import (
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
)

type VacationMonth struct {
	Year  int           `json:"year"`
	Month time.Month    `json:"month"`
	Days  []VacationDay `json:"days"`
}

type VacationCalculationResult struct {
	TotalVacationDays int             `json:"totalVacationDays"`
	Months            []VacationMonth `json:"months"`
}

type VacationDay struct {
	Date        time.Time `json:"date"`
	IsVacation  bool      `json:"isVacation"`  // Входит ли день в отпуск
	Description string    `json:"description"` // Описание дня (если есть в calendar_events)
}

type VacationStats struct {
	Used    int `json:"used"`
	Pending int `json:"pending"`
	Free    int `json:"free"`
}

type VacationCreateRequest struct {
	UserID         string               `json:"userId"`
	StartDate      time.Time            `json:"startDate"`
	EndDate        time.Time            `json:"endDate"`
	Description    string               `json:"description"`
	Status         repo.VacationsStatus `json:"status"`
	VacationTypeID string               `json:"vacationTypeId"`
	// ФИО заявителя — фронт уже знает его (userStore.usersAll), передаёт
	// вместе с заявкой только для текста уведомления админам, нигде не
	// хранится и ни на что не влияет кроме текста.
	ApplicantName string `json:"applicantName"`
}
