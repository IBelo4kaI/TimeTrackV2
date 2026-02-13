package handler

import (
	"timetrack/internal/response"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

func (h *UserHandler) GetUser(c *fiber.Ctx) error {
	token := c.Cookies("session")
	if token == "" {
		return fiber.ErrUnauthorized
	}

	entity := c.Params("entity")
	action := c.Params("action")

	users, err := h.userService.GetUser(c.Context(), token, entity, action)
	if err != nil {
		return err
	}
	return response.Success(c, users)
}
