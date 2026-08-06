package workstandard

import (
	"net/http"
	"strconv"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/response"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// CreateWorkStandard создает новый стандарт работы
func (h *Handler) CreateWorkStandard(c fiber.Ctx) error {
	type request struct {
		UserID        *string `json:"user_id"`
		Month         int32   `json:"month"`
		Year          int32   `json:"year"`
		StandardHours int32   `json:"standard_hours"`
		StandardDays  int32   `json:"standard_days"`
		Gender        int32   `json:"gender"`
	}

	var req request
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, err)
	}

	// Валидация
	if req.Month < 1 || req.Month > 12 {
		return fiber.NewError(http.StatusBadRequest, "month must be between 1 and 12")
	}
	if req.Year < 2000 || req.Year > 2100 {
		return fiber.NewError(http.StatusBadRequest, "year must be between 2000 and 2100")
	}
	if req.StandardHours < 0 {
		return fiber.NewError(http.StatusBadRequest, "standard_hours must be non-negative")
	}
	if req.StandardDays < 0 {
		return fiber.NewError(http.StatusBadRequest, "standard_days must be non-negative")
	}
	if req.Gender < 0 {
		return fiber.NewError(http.StatusBadRequest, "gender must be non-negative")
	}

	params := repo.CreateWorkStandardParams{
		Month:         req.Month,
		Year:          req.Year,
		StandardHours: req.StandardHours,
		StandardDays:  req.StandardDays,
		Gender:        req.Gender,
	}

	// Обработка опционального user_id
	if req.UserID != nil && *req.UserID != "" {
		params.UserID.Valid = true
		params.UserID.String = *req.UserID
	}

	err := h.service.CreateWorkStandard(c.RequestCtx(), params)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Work standard created successfully",
	})
}

// GetWorkStandardById получает стандарт работы по ID
func (h *Handler) GetWorkStandardById(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return fiber.NewError(http.StatusBadRequest, "id is required")
	}

	workStandard, err := h.service.GetWorkStandardById(c.RequestCtx(), id)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, workStandard)
}

// GetWorkStandardsByMonth получает стандарты работы по месяцу и году
func (h *Handler) GetWorkStandardsByMonth(c fiber.Ctx) error {
	monthStr := c.Params("month")
	yearStr := c.Params("year")

	if monthStr == "" || yearStr == "" {
		return fiber.NewError(http.StatusBadRequest, "month and year are required")
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		return fiber.NewError(http.StatusBadRequest, "month must be a number between 1 and 12")
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2000 || year > 2100 {
		return fiber.NewError(http.StatusBadRequest, "year must be a number between 2000 and 2100")
	}

	workStandards, err := h.service.GetWorkStandardsByMonth(c.RequestCtx(), int32(month), int32(year))
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, workStandards)
}

// GetWorkStandardByMonthAndGender получает стандарт работы по месяцу, году, полу и user_id
func (h *Handler) GetWorkStandardByMonthAndGender(c fiber.Ctx) error {
	monthStr := c.Params("month")
	yearStr := c.Params("year")
	genderStr := c.Params("gender")
	userID := c.Query("user_id")

	if monthStr == "" || yearStr == "" || genderStr == "" {
		return fiber.NewError(http.StatusBadRequest, "month, year and gender are required")
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		return fiber.NewError(http.StatusBadRequest, "month must be a number between 1 and 12")
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2000 || year > 2100 {
		return fiber.NewError(http.StatusBadRequest, "year must be a number between 2000 and 2100")
	}

	gender, err := strconv.Atoi(genderStr)
	if err != nil || gender < 0 {
		return fiber.NewError(http.StatusBadRequest, "gender must be a non-negative number")
	}

	workStandard, err := h.service.GetWorkStandardByMonthAndGender(c.RequestCtx(), int32(month), int32(year), int32(gender), userID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, workStandard)
}

// GetWorkStandardsByYear получает стандарты работы по году
func (h *Handler) GetWorkStandardsByYear(c fiber.Ctx) error {
	yearStr := c.Params("year")

	if yearStr == "" {
		return fiber.NewError(http.StatusBadRequest, "year is required")
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2000 || year > 2100 {
		return fiber.NewError(http.StatusBadRequest, "year must be a number between 2000 and 2100")
	}

	workStandards, err := h.service.GetWorkStandardsByYear(c.RequestCtx(), int32(year))
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, workStandards)
}

// GetWorkStandardsByYearGrouped получает стандарты работы по году, сгруппированные по месяцам и полу
func (h *Handler) GetWorkStandardsByYearGrouped(c fiber.Ctx) error {
	yearStr := c.Params("year")

	if yearStr == "" {
		return fiber.NewError(http.StatusBadRequest, "year is required")
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2000 || year > 2100 {
		return fiber.NewError(http.StatusBadRequest, "year must be a number between 2000 and 2100")
	}

	groupedStandards, err := h.service.GetWorkStandardsByYearGrouped(c.RequestCtx(), int32(year))
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, groupedStandards)
}

// UpdateWorkStandard обновляет стандарт работы
func (h *Handler) UpdateWorkStandard(c fiber.Ctx) error {
	type request struct {
		StandardHours int32 `json:"standard_hours"`
		StandardDays  int32 `json:"standard_days"`
	}

	var req request
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, err)
	}

	id := c.Params("id")
	if id == "" {
		return fiber.NewError(http.StatusBadRequest, "id is required")
	}

	if req.StandardHours < 0 {
		return fiber.NewError(http.StatusBadRequest, "standard_hours must be non-negative")
	}
	if req.StandardDays < 0 {
		return fiber.NewError(http.StatusBadRequest, "standard_days must be non-negative")
	}

	params := repo.UpdateWorkStandardParams{
		ID:            id,
		StandardHours: req.StandardHours,
		StandardDays:  req.StandardDays,
	}

	err := h.service.UpdateWorkStandard(c.RequestCtx(), params)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Work standard updated successfully",
	})
}

// DeleteWorkStandard удаляет стандарт работы
func (h *Handler) DeleteWorkStandard(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return fiber.NewError(http.StatusBadRequest, "id is required")
	}

	err := h.service.DeleteWorkStandard(c.RequestCtx(), id)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Work standard deleted successfully",
	})
}
