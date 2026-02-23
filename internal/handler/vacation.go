package handler

import (
	"net/http"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/models"
	"timetrack/internal/response"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v2"
)

type VacationHandler struct {
	service service.VacationService
}

func NewVacationHandler(service service.VacationService) *VacationHandler {
	return &VacationHandler{service: service}
}

func (h *VacationHandler) CreateVacation(c *fiber.Ctx) error {
	var body models.VacationCreateRequest
	if err := c.BodyParser(&body); err != nil {
		return response.Error(c, http.StatusBadRequest, err)
	}

	err := h.service.CreateVacationReport(c.Context(), body)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Created(c)
}

func (h *VacationHandler) GetVacationsByYear(c *fiber.Ctx) error {
	year, err := c.ParamsInt("year")

	if err != nil {
		return response.BadRequest(c)
	}

	userId := c.Params("userId")

	vacations, err := h.service.GetVacationsByYear(c.Context(), userId, year)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, vacations)
}

func (h *VacationHandler) GetAllUserVacationsByYear(c *fiber.Ctx) error {
	year, err := c.ParamsInt("year")

	if err != nil {
		return response.BadRequest(c)
	}

	vacations, err := h.service.GetAllUserVacationsByYear(c.Context(), year)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, vacations)
}

func (h *VacationHandler) CalculateVacationDays(c *fiber.Ctx) error {
	// Получаем параметры из query string
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")

	// Проверяем, что параметры переданы
	if startDateStr == "" || endDateStr == "" {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "Необходимо указать startDate и endDate параметры"))
	}

	// Парсим даты
	const dateLayout = "2006-01-02"
	startDate, err := time.Parse(dateLayout, startDateStr)
	if err != nil {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "Некорректный формат startDate. Используйте YYYY-MM-DD"))
	}

	endDate, err := time.Parse(dateLayout, endDateStr)
	if err != nil {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "Некорректный формат endDate. Используйте YYYY-MM-DD"))
	}

	// Проверяем, что startDate <= endDate
	if startDate.After(endDate) {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "startDate не может быть позже endDate"))
	}

	// Вызываем сервис
	result, err := h.service.CalculateVacationDays(c.Context(), startDate, endDate)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, result)
}

func (h *VacationHandler) GetVacationStatistics(c *fiber.Ctx) error {
	year, err := c.ParamsInt("year")

	if err != nil {
		return response.BadRequest(c)
	}

	userId := c.Params("userId")

	stats, err := h.service.GetVacationsStats(c.Context(), userId, year)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, stats)
}

func (h *VacationHandler) ApproveVacation(c *fiber.Ctx) error {
	vacationID := c.Params("id")

	if vacationID == "" {
		return response.BadRequest(c)
	}

	// Используем UpdateVacationStatus с статусом "approved"
	err := h.service.UpdateVacationStatus(c.Context(), vacationID, repo.VacationsStatusApproved)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Отпуск подтвержден",
	})
}

func (h *VacationHandler) UpdateVacationStatus(c *fiber.Ctx) error {
	vacationID := c.Params("id")

	if vacationID == "" {
		return response.BadRequest(c)
	}

	var body struct {
		Status string `json:"status"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c)
	}

	// Валидация статуса
	var status repo.VacationsStatus
	switch body.Status {
	case "pending":
		status = repo.VacationsStatusPending
	case "approved":
		status = repo.VacationsStatusApproved
	case "rejected":
		status = repo.VacationsStatusRejected
	default:
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "Invalid status. Must be one of: pending, approved, rejected"))
	}

	err := h.service.UpdateVacationStatus(c.Context(), vacationID, status)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Статус обновлен",
	})
}

func (h *VacationHandler) DeleteVacation(c *fiber.Ctx) error {
	vacationID := c.Params("id")

	if vacationID == "" {
		return response.BadRequest(c)
	}

	err := h.service.DeleteVacation(c.Context(), vacationID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Заявка на отпуск удалена",
	})
}
