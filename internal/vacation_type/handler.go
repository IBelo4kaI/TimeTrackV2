package vacationtype

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

func (h Handler) GetVacationTypes(c fiber.Ctx) error {
	types, err := h.service.GetAll(c.RequestCtx())
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}
	return response.Success(c, types)
}

func (h Handler) GetActiveVacationTypes(c fiber.Ctx) error {
	types, err := h.service.GetActive(c.RequestCtx())
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}
	return response.Success(c, types)
}

func (h Handler) CreateVacationType(c fiber.Ctx) error {
	var body CreateVacationTypeRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	vacationType, err := h.service.Create(c.RequestCtx(), body)
	if err != nil {
		return mapError(c, err)
	}

	return response.Success(c, vacationType)
}

func (h Handler) UpdateVacationType(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	var body UpdateVacationTypeRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	vacationType, err := h.service.Update(c.RequestCtx(), id, body)
	if err != nil {
		return mapError(c, err)
	}

	return response.Success(c, vacationType)
}

func (h Handler) DeleteVacationType(c fiber.Ctx) error {
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
	case errors.Is(err, ErrNameTaken), errors.Is(err, ErrTypeInUse):
		return response.Error(c, http.StatusConflict, err)
	case errors.Is(err, ErrColorCodeReq), errors.Is(err, ErrSystemNameEmpty), errors.Is(err, ErrNameEmpty):
		return response.Error(c, http.StatusBadRequest, err)
	default:
		return response.Error(c, http.StatusInternalServerError, err)
	}
}
