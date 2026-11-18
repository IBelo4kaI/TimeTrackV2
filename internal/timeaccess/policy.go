package timeaccess

import (
	"context"
	"errors"
	"strings"
	"timetrack/internal/adapter/permission"
	"timetrack/internal/adapter/reference"
	"timetrack/internal/middleware"
	"timetrack/internal/timeaudit"
	"unicode/utf8"

	"github.com/gofiber/fiber/v3"
)

type Policy struct {
	client *permission.Client
	scope  *reference.DepartmentScopeResolver
	prefix string
}

func NewPolicy(client *permission.Client, scope *reference.DepartmentScopeResolver, prefix string) *Policy {
	return &Policy{client: client, scope: scope, prefix: prefix}
}
func (p *Policy) check(c fiber.Ctx, entity, action, target string) (*permission.PermissionResponse, error) {
	token := c.Cookies(middleware.SessionCookieName)
	if token == "" {
		return nil, fiber.ErrUnauthorized
	}
	result, err := p.client.Validate(c.RequestCtx(), &permission.PermissionRequest{SessionToken: token, Service: p.prefix, Entity: entity, Action: action, UserId: &target})
	if errors.Is(err, permission.ErrUnauthenticated) {
		return nil, fiber.ErrUnauthorized
	}
	if err != nil {
		return nil, fiber.ErrForbidden
	}
	if result == nil || strings.TrimSpace(result.UserId) == "" {
		return nil, fiber.ErrUnauthorized
	}
	return result, nil
}
func (p *Policy) Identity(c fiber.Ctx) (string, error) {
	result, err := p.check(c, "calendar", "read", "")
	if err != nil {
		return "", err
	}
	c.Locals("user_id", result.UserId)
	return result.UserId, nil
}
func (p *Policy) HasAll(c fiber.Ctx, actor, entity, action string) bool {
	result, err := p.check(c, entity+".all", action, "")
	return err == nil && result.IsAccess && result.UserId == actor
}
func (p *Policy) Members(c fiber.Ctx, actor string) (map[string]bool, error) {
	result, err := p.check(c, "department.manager", "manage", "")
	if err != nil {
		return nil, err
	}
	if !result.IsAccess || result.UserId != actor {
		return nil, fiber.ErrForbidden
	}
	members, err := p.scope.Members(c.RequestCtx(), actor, c.Cookies(middleware.SessionCookieName))
	if errors.Is(err, reference.ErrScope) {
		return nil, fiber.ErrForbidden
	}
	if err != nil {
		return nil, fiber.ErrServiceUnavailable
	}
	return members, nil
}

// Authorize keeps existing calendar actions distinct from approval authority.
func (p *Policy) Authorize(c fiber.Ctx, target, action, comment string) (context.Context, error) {
	if strings.TrimSpace(target) == "" {
		return nil, fiber.ErrBadRequest
	}
	entity, permissionTarget := "calendar", target
	if action == "approve" {
		entity, permissionTarget = "timesheet.all", ""
	}
	result, err := p.check(c, entity, action, permissionTarget)
	if err != nil {
		return nil, err
	}
	actor := result.UserId
	c.Locals("user_id", actor)
	if !result.IsAccess {
		c.Locals("time_permission_denied", result)
	}
	role := "admin"
	allowed := result.IsAccess
	if action != "approve" && target == actor {
		role = "employee"
	}
	if !allowed {
		if target == actor {
			return nil, fiber.ErrForbidden
		}
		members, e := p.Members(c, actor)
		if e != nil {
			return nil, e
		}
		if !members[target] {
			return nil, fiber.ErrForbidden
		}
		role = "manager"
	}
	ctx, err := timeaudit.RequestActor(c.RequestCtx(), timeaudit.Actor{ID: actor, Type: "admin", Source: "calendar", Comment: comment})
	if err != nil {
		return nil, fiber.ErrUnauthorized
	}
	if role == "manager" {
		if action != "read" && action != "approve" {
			comment = strings.TrimSpace(comment)
			if comment == "" || utf8.RuneCountInString(comment) > 4000 {
				return nil, fiber.NewError(400, "Требуется комментарий от 1 до 4000 символов")
			}
			return timeaudit.WithManager(ctx, actor, target, comment), nil
		}
		return ctx, nil
	}
	return timeaudit.WithActor(ctx, timeaudit.Actor{ID: actor, Type: role, Source: "calendar", Comment: comment}), nil
}
func (p *Policy) Require(action string, body bool) fiber.Handler {
	return func(c fiber.Ctx) error {
		if c.Cookies(middleware.SessionCookieName) == "" {
			return fiber.ErrUnauthorized
		}
		target, comment := c.Params("userId"), ""
		if body {
			var request map[string]any
			if err := c.Bind().Body(&request); err != nil {
				return fiber.ErrBadRequest
			}
			value, exists := request["userId"]
			if !exists {
				return c.Status(400).JSON(fiber.Map{"message": "userId is required in request body"})
			}
			var ok bool
			target, ok = value.(string)
			if !ok {
				return c.Status(400).JSON(fiber.Map{"message": "userId must be a string"})
			}
			comment, _ = request["comment"].(string)
		}
		ctx, err := p.Authorize(c, target, action, comment)
		if err != nil {
			if errors.Is(err, fiber.ErrForbidden) {
				if denied, ok := c.Locals("time_permission_denied").(*permission.PermissionResponse); ok {
					return c.Status(403).JSON(fiber.Map{"message": denied.Message, "code": denied.Code})
				}
			}
			return err
		}
		c.Locals("time_context", ctx)
		if body {
			c.Request().SetBody(c.Body())
		}
		return c.Next()
	}
}
