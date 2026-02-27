package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/models"
	"timetrack/internal/response"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v2"
)

type UserTimeEntryHandler struct {
	service service.UserTimeEntryService
	logger  *slog.Logger
}

func NewUserTimeEntryHandler(service service.UserTimeEntryService, logger *slog.Logger) *UserTimeEntryHandler {
	return &UserTimeEntryHandler{service: service, logger: logger}
}

func (h *UserTimeEntryHandler) CreateUserTimeEntry(c *fiber.Ctx) error {
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
	if err := c.BodyParser(&body); err != nil {
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

	if err := h.service.CreateUserTimeEntry(c.Context(), prm); err != nil {
		return response.ServerError(c)
	}

	return response.Created(c)
}

func (h *UserTimeEntryHandler) UpdateUserTimeEntries(c *fiber.Ctx) error {
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
	if err := c.BodyParser(&body); err != nil {
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

	if err := h.service.UpdateUserTimeEntries(c.Context(), prm); err != nil {
		return response.ServerError(c)
	}

	return response.Updated(c)
}

func (h *UserTimeEntryHandler) DeleteUserTimeEntries(c *fiber.Ctx) error {
	var prm repo.DeleteUserTimeEntriesParams
	if err := c.BodyParser(&prm); err != nil {
		return response.BadRequest(c)
	}

	if err := h.service.DeleteUserTimeEntries(c.Context(), prm); err != nil {
		return response.ServerError(c)
	}

	return response.Deleted(c)
}

func (h *UserTimeEntryHandler) GetReportStatistics(c *fiber.Ctx) error {
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

	ctx := c.Context()
	// Получаем статистику по часам
	hoursStat, err := h.service.GetStatisticsHoursByMonth(ctx, userId, month, year, gender)
	if err != nil {
		h.logger.Error("Ошибка получения статистики по часам: ",
			slog.String("user_id", userId),
			slog.Int("month", month),
			slog.Int("year", year),
			slog.Int("gender", gender),
			slog.String("error", err.Error()))
		return response.Error(c, http.StatusInternalServerError, err)
	}

	// Получаем статистику по рабочим дням
	workDaysStat, err := h.service.GetStatisticsWorkDaysByMonth(ctx, userId, month, year, gender)
	if err != nil {
		h.logger.Error("Ошибка получения статистики по рабочим дням: ",
			slog.String("user_id", userId),
			slog.Int("month", month),
			slog.Int("year", year),
			slog.Int("gender", gender),
			slog.String("error", err.Error()))
		return response.Error(c, http.StatusInternalServerError, err)
	}

	// Получаем статистику по отпускам (system_name = 'vacation')
	vacationDaysStat, err := h.service.GetCountDaysByMonthWithSystemName(ctx, userId, month, year, gender, "vacation")
	if err != nil {
		h.logger.Error("Ошибка получения статистики по отпускам: ",
			slog.String("user_id", userId),
			slog.Int("month", month),
			slog.Int("year", year),
			slog.Int("gender", gender),
			slog.String("error", err.Error()))
		return response.Error(c, http.StatusInternalServerError, err)
	}

	// Получаем статистику по больничным (предполагаем system_name = 'sick_leave')
	medicalDaysStat, err := h.service.GetCountDaysByMonthWithSystemName(ctx, userId, month, year, gender, "sick_leave")
	if err != nil {
		h.logger.Error("Ошибка получения статистики по больничным: ",
			slog.String("user_id", userId),
			slog.Int("month", month),
			slog.Int("year", year),
			slog.Int("gender", gender),
			slog.String("error", err.Error()))
		return response.Error(c, http.StatusInternalServerError, err)
	}

	// Получаем статистику по отгулам (system_name = 'time-off')
	timeOffDaysStat, err := h.service.GetCountDaysByMonthWithSystemName(ctx, userId, month, year, gender, "time-off")
	if err != nil {
		h.logger.Error("Ошибка получения статистики по отгулам: ",
			slog.String("user_id", userId),
			slog.Int("month", month),
			slog.Int("year", year),
			slog.Int("gender", gender),
			slog.String("error", err.Error()))
		return response.Error(c, http.StatusInternalServerError, err)
	}

	// Получаем статистику по декрету (system_name = 'decree')
	decreeDaysStat, err := h.service.GetCountDaysByMonthWithSystemName(ctx, userId, month, year, gender, "decree")
	if err != nil {
		h.logger.Error("Ошибка получения статистики по декрету: ",
			slog.String("user_id", userId),
			slog.Int("month", month),
			slog.Int("year", year),
			slog.Int("gender", gender),
			slog.String("error", err.Error()))
		return response.Error(c, http.StatusInternalServerError, err)
	}

	responseData := models.ReportStatisticsResponse{
		Hours:        *hoursStat,
		WorkDays:     *workDaysStat,
		VacationDays: *vacationDaysStat,
		MedicalDays:  *medicalDaysStat,
		TimeOffDays:  *timeOffDaysStat,
		DecreeDays:   *decreeDaysStat,
	}

	return response.Success(c, responseData)
}
