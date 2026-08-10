package middleware

import (
	grpcClient "timetrack/internal/adapter/grpc"

	"github.com/gofiber/fiber/v3"
)

type Params struct {
	Service string
	Entity  string
	Action  string
	// RequireAll принудительно требует <entity>.all-разрешение вместо
	// обычного. Нужно для роутов без :userId в пути, которые тем не менее
	// отдают данные НЕСКОЛЬКИХ/любых пользователей (например, "все отпуска
	// за год") — сам permission-сервис добавляет ".all" к entity только
	// когда видит непустой userId в запросе, отличный от userId сессии; без
	// :userId в пути middleware.Require его не узнает и по умолчанию
	// проверил бы только базовое (не .all) разрешение.
	RequireAll bool
}

const SessionCookieName = "session"

// allScopeSentinel — заведомо не совпадающее ни с одним реальным userId
// (session.UserID — это UUID) значение. Передаём его как UserId в
// permission-запрос, чтобы сервис прав всегда добавлял ".all" к entity.
const allScopeSentinel = "all-scope"

func Require(
	client *grpcClient.Client,
	p Params,
) fiber.Handler {
	return func(c fiber.Ctx) error {

		// 1️⃣ достаём session_token
		token := c.Cookies(SessionCookieName)
		if token == "" {
			return fiber.ErrUnauthorized
		}

		userId := c.Params("userId")
		if p.RequireAll {
			userId = allScopeSentinel
		}

		// 3️⃣ gRPC запрос
		resp, err := client.Validate(c.RequestCtx(), &grpcClient.PermissionRequest{
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
	return func(c fiber.Ctx) error {
		// 1️⃣ достаём session_token
		token := c.Cookies(SessionCookieName)
		if token == "" {
			return fiber.ErrUnauthorized
		}

		// 2️⃣ парсим body для получения userId
		var bodyMap map[string]any
		if err := c.Bind().Body(&bodyMap); err != nil {
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
		resp, err := client.Validate(c.RequestCtx(), &grpcClient.PermissionRequest{
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

// RequireOwnerOrAll — для одиночных записей без :userId в пути (например,
// GET /vacation/:id), где владелец записи известен только ПОСЛЕ того, как
// хендлер уже достал её из БД, — обычный Require тут не может заранее
// сравнить userId и не подставляет ".all" сам. Если запись принадлежит
// вызывающему — доступ уже подтверждён базовым Require на этот же роут,
// дополнительно проверять нечего. Если запись чужая — делаем отдельный
// запрос к permission-сервису с UserId = ownerUserID, чтобы получить
// настоящую проверку <entity>.all:<action>.
func RequireOwnerOrAll(
	c fiber.Ctx,
	client *grpcClient.Client,
	p Params,
	callerUserID string,
	ownerUserID string,
) bool {
	if ownerUserID == "" || ownerUserID == callerUserID {
		return true
	}

	token := c.Cookies(SessionCookieName)
	if token == "" {
		return false
	}

	resp, err := client.Validate(c.RequestCtx(), &grpcClient.PermissionRequest{
		SessionToken: token,
		Service:      p.Service,
		Entity:       p.Entity,
		Action:       p.Action,
		UserId:       &ownerUserID,
	})
	if err != nil {
		return false
	}

	return resp.IsAccess
}

// fiber:context-methods migrated
