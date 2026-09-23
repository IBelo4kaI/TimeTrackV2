package smtpsettings

import (
	"timetrack/internal/adapter/grpc"
	"timetrack/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

// SetupRoutes — переиспользует permission-сущность "system_settings"
// (read/edit), как и internal/system_setting — отдельную сущность под
// SMTP заводить не стали.
func SetupRoutes(fiber fiber.Router, service Service, grpc *grpc.Client, prefix string) {
	handler := NewHandler(service)
	router := fiber.Group("/smtp-settings")

	router.Get("",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "system_settings", Action: "read"}),
		handler.GetSettings)

	router.Post("",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "system_settings", Action: "edit"}),
		handler.UpdateSettings)
}
