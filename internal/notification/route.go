package notification

import (
	"timetrack/internal/adapter/grpc"
	"timetrack/internal/middleware"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/sse"
)

func SetupRoutes(f fiber.Router, service Service, grpcClient *grpc.Client, prefix string) {
	handler := NewHandler(service)
	router := f.Group("/notifications")

	require := func(action string) fiber.Handler {
		return middleware.Require(grpcClient, middleware.Params{Service: prefix, Entity: "notification", Action: action})
	}

	// permission notification:read — свои же уведомления, ничего чужого
	router.Get("/stream", require("read"), sse.New(sse.Config{Handler: handler.Stream}))
	router.Get("", require("read"), handler.ListMine)
	router.Get("/unread-count", require("read"), handler.UnreadCount)

	// permission notification:edit
	router.Put("/:id/read", require("edit"), handler.MarkRead)
	router.Put("/read-all", require("edit"), handler.MarkAllRead)

	// permission notification:delete — своя запись, безвозвратно.
	router.Delete("/:id", require("delete"), handler.Delete)
	router.Delete("", require("delete"), handler.DeleteAll)

	// permission notification:create — админская рассылка сотрудникам,
	// не самому себе (в отличие от read/edit/delete — те про свои же).
	router.Post("/send", require("create"), handler.SendManual)
}
