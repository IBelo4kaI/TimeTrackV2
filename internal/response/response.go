package response

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func JSON(c *fiber.Ctx, status int, data any) error {
	return c.Status(status).JSON(data)
}

func Error(c *fiber.Ctx, status int, err error) error {
	return JSON(c, status, map[string]string{"error": err.Error()})
}

func Success(c *fiber.Ctx, data any) error {
	return JSON(c, http.StatusOK, data)
}
