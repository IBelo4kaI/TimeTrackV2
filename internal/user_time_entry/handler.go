package usertimeentry

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/response"
	"timetrack/internal/timeaudit"

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
	ctx, err := timeEntryContext(c)
	if err != nil {
		return fiber.ErrUnauthorized
	}

	type createEntityRequest struct {
		DayTypeID    string    `json:"dayTypeId"`
		HoursWorked  string    `json:"hoursWorked"`
		EntryDate    time.Time `json:"entryDate"`
		WorkLocation *string   `json:"workLocation"`
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
		if !isValidWorkLocation(entity.WorkLocation) {
			return response.BadRequest(c)
		}
		prm = append(prm, repo.CreateUserTimeEntryParams{
			DayTypeID:    entity.DayTypeID,
			HoursWorked:  entity.HoursWorked,
			EntryDate:    entity.EntryDate,
			UserID:       body.UserID,
			WorkLocation: toNullString(entity.WorkLocation),
		})
	}

	if err := h.service.CreateUserTimeEntry(ctx, prm); err != nil {
		return timeEntryError(c, err)
	}

	return response.Created(c)
}

func (h *Handler) UpdateUserTimeEntries(c fiber.Ctx) error {
	ctx, err := timeEntryContext(c)
	if err != nil {
		return fiber.ErrUnauthorized
	}

	type updateEntityRequest struct {
		DayTypeID    string    `json:"dayTypeId"`
		HoursWorked  string    `json:"hoursWorked"`
		EntryDate    time.Time `json:"entryDate"`
		WorkLocation *string   `json:"workLocation"`
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
		if !isValidWorkLocation(entity.WorkLocation) {
			return response.BadRequest(c)
		}
		prm = append(prm, repo.UpdateUserTimeEntryParams{
			DayTypeID:    entity.DayTypeID,
			HoursWorked:  entity.HoursWorked,
			EntryDate:    entity.EntryDate,
			UserID:       body.UserID,
			WorkLocation: toNullString(entity.WorkLocation),
		})
	}

	if err := h.service.UpdateUserTimeEntries(ctx, prm); err != nil {
		return timeEntryError(c, err)
	}

	return response.Updated(c)
}

func (h *Handler) DeleteUserTimeEntries(c fiber.Ctx) error {
	ctx, err := timeEntryContext(c)
	if err != nil {
		return fiber.ErrUnauthorized
	}

	var prm repo.DeleteUserTimeEntriesParams
	if err := c.Bind().Body(&prm); err != nil {
		return response.BadRequest(c)
	}

	if err := h.service.DeleteUserTimeEntries(ctx, prm); err != nil {
		return timeEntryError(c, err)
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

func timeEntryActor(c fiber.Ctx) string {
	id, _ := c.Locals("user_id").(string)
	return id
}

func timeEntryContext(c fiber.Ctx) (context.Context, error) {
	ctx, ok := c.Locals("time_context").(context.Context)
	if !ok {
		return nil, timeaudit.ErrUnauthenticated
	}
	return ctx, timeaudit.Validate(ctx)
}
func timeEntryError(c fiber.Ctx, err error) error {
	if errors.Is(err, timeaudit.ErrProtectedEntry) {
		return fiber.NewError(403, "Руководитель может изменять только рабочие записи")
	}
	if errors.Is(err, timeaudit.ErrUnauthenticated) {
		return fiber.ErrUnauthorized
	}
	return response.ServerError(c)
}
