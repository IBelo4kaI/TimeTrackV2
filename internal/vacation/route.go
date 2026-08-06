package vacation

import (
	"timetrack/internal/adapter/grpc"
	"timetrack/internal/middleware"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(fiber fiber.Router, service Service, fileService *service.FileService, grpc *grpc.Client, prefix string) {
	handler := NewHandler(service, fileService)
	router := fiber.Group("/vacation")

	// расчёт дней отпуска — только авторизация
	router.Get("/calculate", handler.CalculateVacationDays)

	// permission vacation:read
	router.Get("/stats/:userId/:year",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "vacation", Action: "read"}),
		handler.GetVacationStatistics)

	router.Get("/all/:year",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "vacation", Action: "read"}),
		handler.GetAllUserVacationsByYear)

	router.Get("/:userId/:year",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "vacation", Action: "read"}),
		handler.GetVacationsByYear)

	// permission vacation:create
	router.Post("/create",
		middleware.RequireFromBody(grpc, middleware.Params{Service: prefix, Entity: "vacation", Action: "create"}),
		handler.CreateVacation)

	// permission vacation:edit
	router.Put("/:id/approve",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "vacation", Action: "edit"}),
		handler.ApproveVacation)

	router.Put("/:id/status",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "vacation", Action: "edit"}),
		handler.UpdateVacationStatus)

	// File routes for vacations
	router.Post("/:id/file",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "vacation", Action: "edit"}),
		handler.UploadVacationFile)

	router.Get("/file",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "vacation", Action: "read"}),
		handler.GetVacationFile)

	router.Delete("/file",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "vacation", Action: "file_delete"}),
		handler.DeleteVacationFile)

	// permission vacation:delete
	router.Delete("/:id",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "vacation", Action: "delete"}),
		handler.DeleteVacation)
}
