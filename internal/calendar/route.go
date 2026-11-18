package calendar

import (
	"timetrack/internal/adapter/permission"
	"timetrack/internal/timeaccess"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(fiber fiber.Router, service Service, permission *permission.Client, prefix string, policy *timeaccess.Policy) {
	handler := NewHandler(service)
	router := fiber.Group("/calendar")

	// permission calendar.all:read
	router.Get("/:userId/:year/:month",
		policy.Require("read", false),
		handler.GetCalendarDaysWithUserId)
}
