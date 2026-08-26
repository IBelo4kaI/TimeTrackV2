package vk

import (
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

// GenerateLinkCode godoc
// POST /vk/link-code
func (h Handler) GenerateLinkCode(c fiber.Ctx) error {
	code, linkURL := h.service.GenerateLinkCode(callerID(c))
	return response.Success(c, fiber.Map{
		"code":             code,
		"linkUrl":          linkURL,
		"expiresInSeconds": int(LinkCodeTTL.Seconds()),
	})
}

func (h Handler) Unlink(c fiber.Ctx) error {
	if err := h.service.Unlink(c.RequestCtx(), callerID(c)); err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}
	return response.Deleted(c)
}

func (h Handler) Status(c fiber.Ctx) error {
	linked, err := h.service.IsLinked(c.RequestCtx(), callerID(c))
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}
	return response.Success(c, fiber.Map{"linked": linked})
}
