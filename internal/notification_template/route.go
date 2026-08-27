package notificationtemplate

import (
	"timetrack/internal/adapter/grpc"
	"timetrack/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(f fiber.Router, service Service, grpcClient *grpc.Client, prefix string) {
	handler := NewHandler(service)
	router := f.Group("/notification-templates")

	require := func(action string) fiber.Handler {
		return middleware.Require(grpcClient, middleware.Params{Service: prefix, Entity: "notification_templates", Action: action})
	}

	// permission notification_templates:read — админ, выбирающий шаблон для рассылки
	router.Get("", require("read"), handler.GetAll)

	// permission notification_templates:create/edit/delete
	router.Post("", require("create"), handler.Create)
	router.Put("/:id", require("edit"), handler.Update)
	router.Delete("/:id", require("delete"), handler.Delete)
}
