package vacationtype

import (
	"timetrack/internal/adapter/grpc"
	"timetrack/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(fiber fiber.Router, service Service, grpc *grpc.Client, prefix string) {
	handler := NewHandler(service)
	router := fiber.Group("/vacation-types")

	// доступно всем авторизованным — нужно для выбора типа при подаче заявки
	router.Get("", handler.GetVacationTypes)
	router.Get("/active", handler.GetActiveVacationTypes)

	// permission vacation_types:create
	router.Post("",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "vacation_types", Action: "create"}),
		handler.CreateVacationType)

	// permission vacation_types:edit
	router.Put("/:id",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "vacation_types", Action: "edit"}),
		handler.UpdateVacationType)

	// permission vacation_types:delete
	router.Delete("/:id",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "vacation_types", Action: "delete"}),
		handler.DeleteVacationType)
}
