package daytype

import (
	"timetrack/internal/adapter/grpc"
	"timetrack/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(fiber fiber.Router, service Service, grpc *grpc.Client, prefix string) {
	handler := NewHandler(service)
	router := fiber.Group("/daytypes")

	// permission для всех, нужна только авторизация
	router.Get("", handler.GetDayTypes)

	// permission day_types:create
	router.Post("",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "day_types", Action: "create"}),
		handler.CreateDayType)

	// permission day_types:edit
	router.Put("/:id",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "day_types", Action: "edit"}),
		handler.UpdateDayType)

	// permission day_types:delete
	router.Delete("/:id",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "day_types", Action: "delete"}),
		handler.DeleteDayType)
}
