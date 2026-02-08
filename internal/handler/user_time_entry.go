package handler

import (
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/response"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v2"
)

type UserTimeEntryHandler struct {
	service service.UserTimeEntryService
}

func NewUserTimeEntryHandler(service service.UserTimeEntryService) *UserTimeEntryHandler {
	return &UserTimeEntryHandler{service: service}
}

func (h *UserTimeEntryHandler) CreateUserTimeEntry(c *fiber.Ctx) error {
	var entries []repo.CreateUserTimeEntryParams
	if err := c.BodyParser(&entries); err != nil {
		return response.BadRequest(c)
	}

	if err := h.service.CreateUserTimeEntry(c.Context(), entries); err != nil {
		return response.ServerError(c)
	}

	return response.Created(c)
}

func (h *UserTimeEntryHandler) UpdateUserTimeEntries(c *fiber.Ctx) error {
	var prm repo.UpdateUserTimeEntriesParams
	if err := c.BodyParser(&prm); err != nil {
		return response.BadRequest(c)
	}

	if err := h.service.UpdateUserTimeEntries(c.Context(), prm); err != nil {
		return response.ServerError(c)
	}

	return response.Updated(c)
}

func (h *UserTimeEntryHandler) DeleteUserTimeEntries(c *fiber.Ctx) error {
	var ids []string
	if err := c.BodyParser(&ids); err != nil {
		return response.BadRequest(c)
	}

	if err := h.service.DeleteUserTimeEntries(c.Context(), ids); err != nil {
		return response.ServerError(c)
	}

	return response.Deleted(c)
}
