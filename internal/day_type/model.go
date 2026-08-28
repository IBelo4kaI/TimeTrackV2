package daytype

type CreateDayTypeRequest struct {
	Name            string `json:"name"`
	SystemName      string `json:"systemName"`
	ColorCode       string `json:"colorCode"`
	IsWorkDay       bool   `json:"isWorkDay"`
	AffectsVacation bool   `json:"affectsVacation"`
	IsUserSelect    bool   `json:"isUserSelect"`
}

// UpdateDayTypeRequest — без SystemName: он используется в захардкоженных
// местах кода (vacation/sick_leave и т.д.), поэтому неизменяем после создания.
type UpdateDayTypeRequest struct {
	Name            string `json:"name"`
	ColorCode       string `json:"colorCode"`
	IsWorkDay       bool   `json:"isWorkDay"`
	AffectsVacation bool   `json:"affectsVacation"`
	IsUserSelect    bool   `json:"isUserSelect"`
}
