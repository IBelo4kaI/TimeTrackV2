package departmentmanager

import (
	"github.com/gofiber/fiber/v3"
	"timetrack/internal/adapter/permission"
	"timetrack/internal/middleware"
)

func SetupRoutes(router fiber.Router, service *Service, permissions *permission.Client, prefix string) {
	h := &Handler{service: service}
	group := router.Group("/department-managers")
	read := middleware.Require(permissions, middleware.Params{Service: prefix, Entity: "system_settings", Action: "read"})
	edit := middleware.Require(permissions, middleware.Params{Service: prefix, Entity: "system_settings", Action: "edit"})
	group.Get("", read, h.List)
	group.Get("/employees", read, h.Candidates)
	group.Post("", edit, h.Create)
	group.Delete("/:id", edit, h.Delete)
	group.Post("/:id/sync-role", edit, h.SyncRole)
}
