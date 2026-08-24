package notification

import (
	"net/http"
	"strconv"
	"timetrack/internal/response"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return Handler{service: service}
}

func callerID(c fiber.Ctx) string {
	id, _ := c.Locals("user_id").(string)
	return id
}

// ListMine godoc
// GET /notifications?limit=&offset=
func (h Handler) ListMine(c fiber.Ctx) error {
	limit, _ := strconv.ParseInt(c.Query("limit"), 10, 32)
	offset, _ := strconv.ParseInt(c.Query("offset"), 10, 32)

	items, err := h.service.ListMine(c.RequestCtx(), callerID(c), int32(limit), int32(offset))
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}
	return response.Success(c, items)
}

func (h Handler) UnreadCount(c fiber.Ctx) error {
	count, err := h.service.CountUnread(c.RequestCtx(), callerID(c))
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}
	return response.Success(c, fiber.Map{"count": count})
}

func (h Handler) MarkRead(c fiber.Ctx) error {
	if err := h.service.MarkRead(c.RequestCtx(), c.Params("id"), callerID(c)); err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}
	return response.Updated(c)
}

func (h Handler) MarkAllRead(c fiber.Ctx) error {
	if err := h.service.MarkAllRead(c.RequestCtx(), callerID(c)); err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}
	return response.Updated(c)
}
