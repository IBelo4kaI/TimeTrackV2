package handler

import (
	"net/http"
	"time"
	"timetrack/internal/response"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v2"
)

const calendarEventDateLayout = "2006-01-02"

type CalendarEventHandler struct {
	service service.CalendarEventService
}

func NewCalendarEventHandler(svc service.CalendarEventService) *CalendarEventHandler {
	return &CalendarEventHandler{service: svc}
}

func (h *CalendarEventHandler) GetCalendarEventsForYear(c *fiber.Ctx) error {
	year, err := c.ParamsInt("year")
	if err != nil {
		return response.BadRequest(c)
	}

	events, err := h.service.GetCalendarEventsForYear(c.Context(), year)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, events)
}

func (h *CalendarEventHandler) GetCalendarEventsForMonth(c *fiber.Ctx) error {
	year, err := c.ParamsInt("year")
	if err != nil {
		return response.BadRequest(c)
	}

	month, err := c.ParamsInt("month")
	if err != nil {
		return response.BadRequest(c)
	}

	events, err := h.service.GetCalendarEventsForMonth(c.Context(), month, year)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, events)
}

func (h *CalendarEventHandler) GetCalendarEventByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	event, err := h.service.GetCalendarEventByID(c.Context(), id)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, event)
}

func (h *CalendarEventHandler) CreateCalendarEvent(c *fiber.Ctx) error {
	var body struct {
		EventDate   string `json:"eventDate"`
		DayTypeID   string `json:"dayTypeId"`
		Description string `json:"description"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.Error(c, http.StatusBadRequest, err)
	}

	if body.EventDate == "" || body.DayTypeID == "" {
		return response.BadRequest(c)
	}

	eventDate, err := time.Parse(calendarEventDateLayout, body.EventDate)
	if err != nil {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "Некорректный формат eventDate. Используйте YYYY-MM-DD"))
	}

	if err := h.service.CreateCalendarEvent(c.Context(), eventDate, body.DayTypeID, body.Description); err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Created(c)
}

func (h *CalendarEventHandler) UpdateCalendarEvent(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	var body struct {
		EventDate   string `json:"eventDate"`
		DayTypeID   string `json:"dayTypeId"`
		Description string `json:"description"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.Error(c, http.StatusBadRequest, err)
	}

	if body.EventDate == "" || body.DayTypeID == "" {
		return response.BadRequest(c)
	}

	eventDate, err := time.Parse(calendarEventDateLayout, body.EventDate)
	if err != nil {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "Некорректный формат eventDate. Используйте YYYY-MM-DD"))
	}

	if err := h.service.UpdateCalendarEvent(c.Context(), id, eventDate, body.DayTypeID, body.Description); err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Updated(c)
}

func (h *CalendarEventHandler) DeleteCalendarEvent(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	if err := h.service.DeleteCalendarEvent(c.Context(), id); err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Deleted(c)
}
