package handler

import (
	"time"
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
	type createEntityRequest struct {
		DayTypeID   string    `json:"dayTypeId"`
		HoursWorked string    `json:"hoursWorked"`
		EntryDate   time.Time `json:"entryDate"`
	}
	type createRequest struct {
		UserID   string                `json:"userId"`
		Entities []createEntityRequest `json:"entities"`
	}
	var body createRequest
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c)
	}

	var prm []repo.CreateUserTimeEntryParams

	for _, entity := range body.Entities {
		prm = append(prm, repo.CreateUserTimeEntryParams{
			DayTypeID:   entity.DayTypeID,
			HoursWorked: entity.HoursWorked,
			EntryDate:   entity.EntryDate,
			UserID:      body.UserID,
		})
	}

	if err := h.service.CreateUserTimeEntry(c.Context(), prm); err != nil {
		return response.ServerError(c)
	}

	return response.Created(c)
}

func (h *UserTimeEntryHandler) UpdateUserTimeEntries(c *fiber.Ctx) error {
	type updateEntityRequest struct {
		DayTypeID   string    `json:"dayTypeId"`
		HoursWorked string    `json:"hoursWorked"`
		EntryDate   time.Time `json:"entryDate"`
	}
	type updateRequest struct {
		UserID   string                `json:"userId"`
		Entities []updateEntityRequest `json:"entities"`
	}
	var body updateRequest
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c)
	}

	var prm []repo.UpdateUserTimeEntryParams

	for _, entity := range body.Entities {
		prm = append(prm, repo.UpdateUserTimeEntryParams{
			DayTypeID:   entity.DayTypeID,
			HoursWorked: entity.HoursWorked,
			EntryDate:   entity.EntryDate,
			UserID:      body.UserID,
		})
	}

	if err := h.service.UpdateUserTimeEntries(c.Context(), prm); err != nil {
		return response.ServerError(c)
	}

	return response.Updated(c)
}

func (h *UserTimeEntryHandler) DeleteUserTimeEntries(c *fiber.Ctx) error {
	var prm repo.DeleteUserTimeEntriesParams
	if err := c.BodyParser(&prm); err != nil {
		return response.BadRequest(c)
	}

	if err := h.service.DeleteUserTimeEntries(c.Context(), prm); err != nil {
		return response.ServerError(c)
	}

	return response.Deleted(c)
}
