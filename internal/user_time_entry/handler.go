package usertimeentry

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/response"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	service Service
	logger  *slog.Logger
}

func NewHandler(service Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) CreateUserTimeEntry(c fiber.Ctx) error {
	type createEntityRequest struct {
		DayTypeID   string    `json:"dayTypeId"`
		HoursWorked string    `json:"hoursWorked"`
		EntryDate   time.Time `json:"entryDate"`
	}
	type createRequest struct {
		UserID   string                `json:"userId"`
		Entities []createEntityRequest `json:"entities"`
	}
	var body createRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	var prm []repo.CreateUserTimeEntryParams

	for _, entity := range body.Entities {
		prm = append(prm, repo.CreateUserTimeEntryParams{
			DayTypeID:   entity.DayTypeID,
			HoursWorked: entity.HoursWorked,
			EntryDate:   entity.EntryDate,
			UserID:      body.UserID,
		})
	}

	if err := h.service.CreateUserTimeEntry(c.RequestCtx(), prm); err != nil {
		return response.ServerError(c)
	}

	return response.Created(c)
}

func (h *Handler) UpdateUserTimeEntries(c fiber.Ctx) error {
	type updateEntityRequest struct {
		DayTypeID   string    `json:"dayTypeId"`
		HoursWorked string    `json:"hoursWorked"`
		EntryDate   time.Time `json:"entryDate"`
	}
	type updateRequest struct {
		UserID   string                `json:"userId"`
		Entities []updateEntityRequest `json:"entities"`
	}
	var body updateRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	var prm []repo.UpdateUserTimeEntryParams

	for _, entity := range body.Entities {
		prm = append(prm, repo.UpdateUserTimeEntryParams{
			DayTypeID:   entity.DayTypeID,
			HoursWorked: entity.HoursWorked,
			EntryDate:   entity.EntryDate,
			UserID:      body.UserID,
		})
	}

	if err := h.service.UpdateUserTimeEntries(c.RequestCtx(), prm); err != nil {
		return response.ServerError(c)
	}

	return response.Updated(c)
}

func (h *Handler) DeleteUserTimeEntries(c fiber.Ctx) error {
	var prm repo.DeleteUserTimeEntriesParams
	if err := c.Bind().Body(&prm); err != nil {
		return response.BadRequest(c)
	}

	if err := h.service.DeleteUserTimeEntries(c.RequestCtx(), prm); err != nil {
		return response.ServerError(c)
	}

	return response.Deleted(c)
}

func (h *Handler) GetReportStatistics(c fiber.Ctx) error {
	userId := c.Params("userId")
	yearStr := c.Params("year")
	monthStr := c.Params("month")
	genderStr := c.Params("gender")

	if userId == "" || monthStr == "" || yearStr == "" || genderStr == "" {
		return response.BadRequest(c)
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil {
		return response.BadRequest(c)
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return response.BadRequest(c)
	}

	gender, err := strconv.Atoi(genderStr)
	if err != nil {
		return response.BadRequest(c)
	}

	stat, err := h.service.GetReportStatistics(c.RequestCtx(), userId, month, year, gender)
	if err != nil {
		h.logger.Error("Ошибка получения статистики: ",
			slog.String("user_id", userId),
			slog.Int("month", month),
			slog.Int("year", year),
			slog.Int("gender", gender),
			slog.String("error", err.Error()))
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, stat)
}
