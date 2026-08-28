package news

import (
	"timetrack/internal/adapter/grpc"
	"timetrack/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(f fiber.Router, service Service, grpcClient *grpc.Client, prefix string) {
	handler := NewHandler(service)
	router := f.Group("/news")

	// доступно всем авторизованным — чейнджлог общий для всех сотрудников
	router.Get("", handler.List)
	router.Get("/unread-count", handler.UnreadCount)
	router.Post("/mark-seen", handler.MarkSeen)

	// permission news:create/edit/delete — только админам
	router.Post("",
		middleware.Require(grpcClient, middleware.Params{Service: prefix, Entity: "news", Action: "create"}),
		handler.Create)
	router.Put("/:id",
		middleware.Require(grpcClient, middleware.Params{Service: prefix, Entity: "news", Action: "edit"}),
		handler.Update)
	router.Delete("/:id",
		middleware.Require(grpcClient, middleware.Params{Service: prefix, Entity: "news", Action: "delete"}),
		handler.Delete)
}
