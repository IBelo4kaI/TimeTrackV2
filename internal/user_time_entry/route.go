package usertimeentry

import (
	"log/slog"
	"timetrack/internal/adapter/permission"
	"timetrack/internal/timeaccess"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(fiber fiber.Router, service Service, logger *slog.Logger, permission *permission.Client, prefix string, policy *timeaccess.Policy) {
	handler := NewHandler(service, logger)
	router := fiber.Group("/usertimeentries")

	// permission usertime:edit
	router.Post("/create",
		policy.Require("create", true),
		handler.CreateUserTimeEntry)

	router.Post("/update",
		policy.Require("edit", true),
		handler.UpdateUserTimeEntries)

	router.Post("/delete",
		policy.Require("delete", true),
		handler.DeleteUserTimeEntries)

	// Report statistics route
	router.Get("/statistics/:userId/:year/:month/:gender",
		policy.Require("read", false),
		handler.GetReportStatistics)

}
