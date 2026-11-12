package usertimeentry

import (
	"log/slog"
	"timetrack/internal/adapter/permission"
	"timetrack/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(fiber fiber.Router, service Service, logger *slog.Logger, permission *permission.Client, prefix string) {
	handler := NewHandler(service, logger)
	router := fiber.Group("/usertimeentries")

	// permission usertime:edit
	router.Post("/create",
		middleware.RequireFromBody(permission, middleware.Params{Service: prefix, Entity: "calendar", Action: "create"}),
		handler.CreateUserTimeEntry)

	router.Post("/update",
		middleware.RequireFromBody(permission, middleware.Params{Service: prefix, Entity: "calendar", Action: "edit"}),
		handler.UpdateUserTimeEntries)

	router.Post("/delete",
		middleware.RequireFromBody(permission, middleware.Params{Service: prefix, Entity: "calendar", Action: "delete"}),
		handler.DeleteUserTimeEntries)

	// Report statistics route
	router.Get("/statistics/:userId/:year/:month/:gender",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "calendar", Action: "read"}),
		handler.GetReportStatistics)

}
