package filecategory

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

func (h Handler) GetFileCategories(c fiber.Ctx) error {
	tree, err := h.service.GetTree(c.RequestCtx())
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}
	return response.Success(c, tree)
}

func (h Handler) CreateFileCategory(c fiber.Ctx) error {
	var body CreateFileCategoryRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}
	if body.Name == "" {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "название категории обязательно"))
	}

	category, err := h.service.Create(c.RequestCtx(), body)
	if err != nil {
		return mapError(c, err)
	}

	return response.Success(c, category)
}

func (h Handler) UpdateFileCategory(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	var body UpdateFileCategoryRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}
	if body.Name == "" {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "название категории обязательно"))
	}

	category, err := h.service.Update(c.RequestCtx(), id, body)
	if err != nil {
		return mapError(c, err)
	}

	return response.Success(c, category)
}

func (h Handler) DeleteFileCategory(c fiber.Ctx) error {
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
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrParentMissing):
		return response.Error(c, http.StatusNotFound, err)
	case errors.Is(err, ErrNameTaken), errors.Is(err, ErrCycle):
		return response.Error(c, http.StatusConflict, err)
	case errors.Is(err, ErrSystemCategory):
		return response.Error(c, http.StatusForbidden, err)
	default:
		return response.Error(c, http.StatusInternalServerError, err)
	}
}
