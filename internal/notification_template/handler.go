package notificationtemplate

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

func (h Handler) GetAll(c fiber.Ctx) error {
	templates, err := h.service.GetAll(c.RequestCtx())
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}
	return response.Success(c, templates)
}

func (h Handler) Create(c fiber.Ctx) error {
	var body CreateNotificationTemplateRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	template, err := h.service.Create(c.RequestCtx(), body)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, template)
}

func (h Handler) Update(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	var body UpdateNotificationTemplateRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	template, err := h.service.Update(c.RequestCtx(), id, body)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, template)
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
	case errors.Is(err, ErrNameTaken):
		return response.Error(c, http.StatusConflict, err)
	case errors.Is(err, ErrNameEmpty), errors.Is(err, ErrTitleReq):
		return response.Error(c, http.StatusBadRequest, err)
	default:
		return response.Error(c, http.StatusInternalServerError, err)
	}
}
