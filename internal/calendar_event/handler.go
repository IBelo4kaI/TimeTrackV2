package calendarevent

import (
	"net/http"
	"time"
	"timetrack/internal/response"

	"github.com/gofiber/fiber/v3"
)

const calendarEventDateLayout = "2006-01-02"

type Handler struct {
	service Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) GetCalendarEventsForYear(c fiber.Ctx) error {
	year, err := fiber.Params[int](c, "year"), error(nil)
	if err != nil {
		return response.BadRequest(c)
	}

	events, err := h.service.GetCalendarEventsForYear(c.RequestCtx(), year)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, events)
}

func (h *Handler) GetCalendarEventsForMonth(c fiber.Ctx) error {
	year, err := fiber.Params[int](c, "year"), error(nil)
	if err != nil {
		return response.BadRequest(c)
	}

	month, err := fiber.Params[int](c, "month"), error(nil)
	if err != nil {
		return response.BadRequest(c)
	}

	events, err := h.service.GetCalendarEventsForMonth(c.RequestCtx(), month, year)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, events)
}

func (h *Handler) GetCalendarEventByID(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	event, err := h.service.GetCalendarEventByID(c.RequestCtx(), id)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, event)
}

func (h *Handler) CreateCalendarEvent(c fiber.Ctx) error {
	var body struct {
		EventDate   string `json:"eventDate"`
		DayTypeID   string `json:"dayTypeId"`
		Description string `json:"description"`
	}

	if err := c.Bind().Body(&body); err != nil {
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

	if err := h.service.CreateCalendarEvent(c.RequestCtx(), eventDate, body.DayTypeID, body.Description); err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Created(c)
}

func (h *Handler) UpdateCalendarEvent(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	var body struct {
		EventDate   string `json:"eventDate"`
		DayTypeID   string `json:"dayTypeId"`
		Description string `json:"description"`
	}

	if err := c.Bind().Body(&body); err != nil {
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

	if err := h.service.UpdateCalendarEvent(c.RequestCtx(), id, eventDate, body.DayTypeID, body.Description); err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Updated(c)
}

func (h *Handler) DeleteCalendarEvent(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	if err := h.service.DeleteCalendarEvent(c.RequestCtx(), id); err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Deleted(c)
}
