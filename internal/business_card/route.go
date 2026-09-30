package businesscard

import (
	"timetrack/internal/adapter/grpc"
	"timetrack/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(r fiber.Router, service Service, grpc *grpc.Client, prefix string) {
	handler := NewHandler(service, grpc, prefix)
	router := r.Group("/business-cards")

	entity := "business_cards"
	all := func(action string) fiber.Handler {
		return middleware.Require(grpc, middleware.Params{Service: prefix, Entity: entity, Action: action, RequireAll: true})
	}

	// /my и /all — до /:id, иначе Fiber принял бы их за :id
	router.Get("/my",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: entity, Action: "read"}),
		handler.GetMy)
	router.Get("/all", all("read"), handler.GetAll)

	router.Post("/", all("create"), handler.Create)

	// владелец довалидируется в хендлере через RequireOwnerOrAll
	router.Get("/:id",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: entity, Action: "read"}),
		handler.GetCard)

	router.Put("/:id", all("edit"), handler.Update)
	router.Delete("/:id", all("delete"), handler.Delete)
	router.Get("/:id/number", all("read"), handler.GetNumber)
	router.Get("/:id/history", all("read"), handler.GetHistory)
	router.Post("/:id/assign", all("edit"), handler.Assign)
	router.Post("/:id/release", all("edit"), handler.Release)
}
