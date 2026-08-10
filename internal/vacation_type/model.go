package vacationtype

type CreateVacationTypeRequest struct {
	Name           string `json:"name"`
	SystemName     string `json:"systemName"`
	ColorCode      string `json:"colorCode"`
	AffectsBalance bool   `json:"affectsBalance"`
	IsActive       bool   `json:"isActive"`
	SortOrder      int32  `json:"sortOrder"`
}

type UpdateVacationTypeRequest struct {
	Name           string `json:"name"`
	SystemName     string `json:"systemName"`
	ColorCode      string `json:"colorCode"`
	AffectsBalance bool   `json:"affectsBalance"`
	IsActive       bool   `json:"isActive"`
	SortOrder      int32  `json:"sortOrder"`
}
