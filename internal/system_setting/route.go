package systemsetting

import (
	"timetrack/internal/adapter/permission"
	"timetrack/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(fiber fiber.Router, service Service, permission *permission.Client, prefix string) {
	handler := NewHandler(service)
	router := fiber.Group("/system-settings")

	// permission system_settings:read
	router.Get("/:key",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "system_settings", Action: "read"}),
		handler.GetSystemSettingByKey)

	// permission system_settings:edit
	router.Post("/value",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "system_settings", Action: "edit"}),
		handler.UpdateSystemSettingValue)
}
