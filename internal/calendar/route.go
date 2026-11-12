package calendar

import (
	"timetrack/internal/adapter/permission"
	"timetrack/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(fiber fiber.Router, service Service, permission *permission.Client, prefix string) {
	handler := NewHandler(service)
	router := fiber.Group("/calendar")

	// permission calendar.all:read
	router.Get("/:userId/:year/:month",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "calendar", Action: "read"}),
		handler.GetCalendarDaysWithUserId)
}
