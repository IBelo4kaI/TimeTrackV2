package news

import (
	"errors"
	"net/http"
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

func (h Handler) List(c fiber.Ctx) error {
	items, err := h.service.GetAll(c.RequestCtx())
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

func (h Handler) MarkSeen(c fiber.Ctx) error {
	if err := h.service.MarkSeen(c.RequestCtx(), callerID(c)); err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}
	return response.Updated(c)
}

func (h Handler) Create(c fiber.Ctx) error {
	var body CreatePostRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	post, err := h.service.Create(c.RequestCtx(), body)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, post)
}

func (h Handler) Update(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	var body UpdatePostRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	post, err := h.service.Update(c.RequestCtx(), id, body)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, post)
}

func (h Handler) Delete(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	if err := h.service.Delete(c.RequestCtx(), id); err != nil {
		return mapError(c, err)
	}
	return response.Deleted(c)
}

func mapError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return response.Error(c, http.StatusNotFound, err)
	case errors.Is(err, ErrTitleReq), errors.Is(err, ErrBodyReq):
		return response.Error(c, http.StatusBadRequest, err)
	default:
		return response.Error(c, http.StatusInternalServerError, err)
	}
}
