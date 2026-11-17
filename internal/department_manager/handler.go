package departmentmanager

import (
	"database/sql"
	"errors"
	"github.com/gofiber/fiber/v3"
	"timetrack/internal/middleware"
	"timetrack/internal/response"
)

type Handler struct{ service *Service }

func replyError(c fiber.Ctx, err error) error {
	var fe *fiber.Error
	if errors.As(err, &fe) {
		return response.Error(c, fe.Code, err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return response.Error(c, 404, errors.New("Назначение руководителя не найдено"))
	}
	return response.ServerError(c)
}
func (h *Handler) List(c fiber.Ctx) error {
	rows, err := h.service.List(c.RequestCtx())
	if err != nil {
		return replyError(c, err)
	}
	c.Set("Cache-Control", "no-store")
	return response.Success(c, rows)
}
func (h *Handler) Candidates(c fiber.Ctx) error {
	rows, err := h.service.Candidates(c.RequestCtx(), c.Cookies(middleware.SessionCookieName))
	if err != nil {
		return replyError(c, err)
	}
	c.Set("Cache-Control", "no-store")
	return response.Success(c, rows)
}
func (h *Handler) Create(c fiber.Ctx) error {
	var body AssignmentRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}
	actor, _ := c.Locals("user_id").(string)
	row, err := h.service.Create(c.RequestCtx(), body, actor, c.Cookies(middleware.SessionCookieName))
	if err != nil {
		return replyError(c, err)
	}
	return c.Status(201).JSON(row)
}
func (h *Handler) Delete(c fiber.Ctx) error {
	if err := h.service.Delete(c.RequestCtx(), c.Params("id")); err != nil {
		return replyError(c, err)
	}
	return response.Deleted(c)
}
func (h *Handler) SyncRole(c fiber.Ctx) error {
	row, err := h.service.SyncRole(c.RequestCtx(), c.Params("id"))
	if err != nil {
		return replyError(c, err)
	}
	return response.Success(c, row)
}
