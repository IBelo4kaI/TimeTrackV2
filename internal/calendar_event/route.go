package calendarevent

import (
	"timetrack/internal/adapter/grpc"
	"timetrack/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(fiber fiber.Router, service Service, grpc *grpc.Client, prefix string) {
	handler := NewHandler(service)
	router := fiber.Group("/calendar-events")

	// permission calendar_events:read
	router.Get("/year/:year/month/:month",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "calendar_events", Action: "read"}),
		handler.GetCalendarEventsForMonth)

	router.Get("/year/:year",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "calendar_events", Action: "read"}),
		handler.GetCalendarEventsForYear)

	router.Get("/:id",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "calendar_events", Action: "read"}),
		handler.GetCalendarEventByID)

	// permission calendar_events:create
	router.Post("",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "calendar_events", Action: "create"}),
		handler.CreateCalendarEvent)

	// permission calendar_events:edit
	router.Put("/:id",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "calendar_events", Action: "edit"}),
		handler.UpdateCalendarEvent)

	// permission calendar_events:delete
	router.Delete("/:id",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "calendar_events", Action: "delete"}),
		handler.DeleteCalendarEvent)
}
