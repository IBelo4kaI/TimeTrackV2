package workstandard

import (
	"timetrack/internal/adapter/grpc"
	"timetrack/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(fiber fiber.Router, service Service, grpc *grpc.Client, prefix string) {
	handler := NewHandler(service)
	router := fiber.Group("/work-standards")

	// permission work_standards:create
	router.Post("",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "work_standards", Action: "create"}),
		handler.CreateWorkStandard)

	// permission work_standards:read
	router.Get("/month/:month/year/:year",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "work_standards", Action: "read"}),
		handler.GetWorkStandardsByMonth)

	router.Get("/year/:year",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "work_standards", Action: "read"}),
		handler.GetWorkStandardsByYear)

	router.Get("/year/:year/grouped",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "work_standards", Action: "read"}),
		handler.GetWorkStandardsByYearGrouped)

	// permission work_standards:edit
	router.Put("/:id",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "work_standards", Action: "edit"}),
		handler.UpdateWorkStandard)

	// permission work_standards:delete
	router.Delete("/:id",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "work_standards", Action: "delete"}),
		handler.DeleteWorkStandard)
}
