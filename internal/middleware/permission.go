package middleware

import (
	grpcClient "timetrack/internal/adapter/grpc"

	"github.com/gofiber/fiber/v2"
)

type Params struct {
	Service string
	Entity  string
	Action  string
}

const SessionCookieName = "session"

func Require(
	client *grpcClient.Client,
	p Params,
) fiber.Handler {
	return func(c *fiber.Ctx) error {

		// 1️⃣ достаём session_token
		token := c.Cookies(SessionCookieName)
		if token == "" {
			return fiber.ErrUnauthorized
		}

		userId := c.Params("userId")

		// 3️⃣ gRPC запрос
		resp, err := client.Validate(c.Context(), &grpcClient.PermissionRequest{
			SessionToken: token,
			Service:      p.Service,
			Entity:       p.Entity,
			Action:       p.Action,
			UserId:       &userId,
		})

		if err != nil {
			// auth/permission сервис недоступен → deny by default
			return fiber.ErrForbidden
		}

		if !resp.IsAccess {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"message": resp.Message,
				"code":    resp.Code,
			})
		}

		c.Locals("user_id", resp.UserId)

		return c.Next()
	}
}

func RequireFromBody(
	client *grpcClient.Client,
	p Params,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1️⃣ достаём session_token
		token := c.Cookies(SessionCookieName)
		if token == "" {
			return fiber.ErrUnauthorized
		}

		// 2️⃣ парсим body для получения userId
		var bodyMap map[string]any
		if err := c.BodyParser(&bodyMap); err != nil {
			return fiber.ErrBadRequest
		}

		userIdVal, ok := bodyMap["userId"]
		if !ok {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "userId is required in request body",
			})
		}

		userId, ok := userIdVal.(string)
		if !ok {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "userId must be a string",
			})
		}

		// 4️⃣ gRPC запрос
		resp, err := client.Validate(c.Context(), &grpcClient.PermissionRequest{
			SessionToken: token,
			Service:      p.Service,
			Entity:       p.Entity,
			Action:       p.Action,
			UserId:       &userId,
		})

		if err != nil {
			return fiber.ErrForbidden
		}

		if !resp.IsAccess {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"message": resp.Message,
				"code":    resp.Code,
			})
		}

		c.Locals("user_id", resp.UserId)

		// 5️⃣ ВАЖНО: восстанавливаем body для следующего handler
		c.Request().SetBody(c.Body())

		return c.Next()
	}

}
