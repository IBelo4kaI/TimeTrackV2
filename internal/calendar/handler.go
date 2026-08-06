package calendar

import (
	"net/http"
	"timetrack/internal/response"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetCalendarDaysWithUserId(c fiber.Ctx) error {
	userId := c.Params("userId")
	month, err := fiber.Params[int](c, "month"), error(nil)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, err)
	}
	year, err := fiber.Params[int](c, "year"), error(nil)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, err)
	}
	calendarDays, err := h.service.GetCalendarDays(c.RequestCtx(), userId, month, year)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, calendarDays)
}

func (h *Handler) GetCalendarDays(c fiber.Ctx) error {
	userId := c.Params("userId")
	month, err := fiber.Params[int](c, "month"), error(nil)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, err)
	}
	year, err := fiber.Params[int](c, "year"), error(nil)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, err)
	}

	calendarDays, err := h.service.GetCalendarDays(c.RequestCtx(), userId, month, year)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, calendarDays)
}
