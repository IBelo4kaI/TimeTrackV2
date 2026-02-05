package handler

import (
	"net/http"
	"timetrack/internal/response"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v2"
)

type CalendarHandler struct {
	service service.CalendarService
}

func NewCalendarHandler(service service.CalendarService) *CalendarHandler {
	return &CalendarHandler{
		service: service,
	}
}

func (h *CalendarHandler) GetCalendarDaysWithUserId(c *fiber.Ctx) error {
	userId := c.Params("userId")
	month, err := c.ParamsInt("month")
	if err != nil {
		return response.Error(c, http.StatusBadRequest, err)
	}
	year, err := c.ParamsInt("year")
	if err != nil {
		return response.Error(c, http.StatusBadRequest, err)
	}

	calendarDays, err := h.service.GetCalendarDays(c.Context(), userId, month, year)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, calendarDays)
}

func (h *CalendarHandler) GetCalendarDays(c *fiber.Ctx) error {
	userId := c.Params("userId")
	month, err := c.ParamsInt("month")
	if err != nil {
		return response.Error(c, http.StatusBadRequest, err)
	}
	year, err := c.ParamsInt("year")
	if err != nil {
		return response.Error(c, http.StatusBadRequest, err)
	}

	calendarDays, err := h.service.GetCalendarDays(c.Context(), userId, month, year)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, calendarDays)
}
