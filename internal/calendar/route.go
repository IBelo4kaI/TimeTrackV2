package calendar

import (
	"timetrack/internal/adapter/grpc"
	"timetrack/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(fiber fiber.Router, service Service, grpc *grpc.Client, prefix string) {
	handler := NewHandler(service)
	router := fiber.Group("/calendar")

	// permission calendar.all:read
	router.Get("/:userId/:year/:month",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "calendar", Action: "read"}),
		handler.GetCalendarDaysWithUserId)
}
