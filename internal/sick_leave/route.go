package sickleave

import (
	"timetrack/internal/adapter/grpc"
	"timetrack/internal/middleware"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(fiber fiber.Router, service Service, fileService *service.FileService, grpc *grpc.Client, prefix string) {
	handler := NewHandler(service, fileService, grpc, prefix)
	router := fiber.Group("/sick-leaves")

	// permission sick_leaves:create
	router.Post("/create",
		middleware.RequireFromBody(grpc, middleware.Params{Service: prefix, Entity: "sick_leaves", Action: "create"}),
		handler.CreateSickLeave)

	// permission sick_leaves:read
	router.Get("/stats/:userId/:year",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "sick_leaves", Action: "read"}),
		handler.GetSickLeaveStatistics)

	// permission sick_leaves:read
	// эндпоинт отдаёт больничные ВСЕХ сотрудников — без RequireAll middleware
	// проверил бы обычный "sick_leaves:read" вместо "sick_leaves.all:read"
	router.Get("/all/:year",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "sick_leaves", Action: "read", RequireAll: true}),
		handler.GetAllUsersSickLeavesByYear)

	router.Get("/:userId/:year",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "sick_leaves", Action: "read"}),
		handler.GetSickLeavesByYear)

	// статус официальный/неофициальный ставит только тот, кто управляет чужими
	// больничными (sick_leaves.all:edit) — не владелец по базовому edit
	router.Put("/:id/status",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "sick_leaves", Action: "edit", RequireAll: true}),
		handler.UpdateSickLeaveStatus)

	// загрузка файла (свой — sick_leaves:edit, чужой — sick_leaves.all:edit,
	// владелец проверяется в хендлере); просмотр: GET /v1/files/entity/sick_leave/:id, удаление: DELETE /v1/files/:id
	router.Post("/:id/file",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "sick_leaves", Action: "edit"}),
		handler.UploadSickLeaveFile)

	// permission sick_leaves:delete (чужой — sick_leaves.all:delete, владелец
	// проверяется в хендлере)
	router.Delete("/:id",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "sick_leaves", Action: "delete"}),
		handler.DeleteSickLeave)
}
