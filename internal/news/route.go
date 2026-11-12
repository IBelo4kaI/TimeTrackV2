package news

import (
	"timetrack/internal/adapter/permission"
	"timetrack/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(f fiber.Router, service Service, permissionClient *permission.Client, prefix string) {
	handler := NewHandler(service)
	router := f.Group("/news")

	// доступно всем авторизованным — чейнджлог общий для всех сотрудников
	router.Get("", handler.List)
	router.Get("/unread-count", handler.UnreadCount)
	router.Post("/mark-seen", handler.MarkSeen)

	// permission news:create/edit/delete — только админам
	router.Post("",
		middleware.Require(permissionClient, middleware.Params{Service: prefix, Entity: "news", Action: "create"}),
		handler.Create)
	router.Put("/:id",
		middleware.Require(permissionClient, middleware.Params{Service: prefix, Entity: "news", Action: "edit"}),
		handler.Update)
	router.Delete("/:id",
		middleware.Require(permissionClient, middleware.Params{Service: prefix, Entity: "news", Action: "delete"}),
		handler.Delete)
}
