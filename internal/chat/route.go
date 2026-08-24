package chat

import (
	"timetrack/internal/adapter/grpc"
	"timetrack/internal/middleware"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/sse"
)

func SetupRoutes(f fiber.Router, service Service, grpcClient *grpc.Client, prefix string) {
	handler := NewHandler(service, grpcClient, prefix)
	router := f.Group("/chats")

	require := func(action string) fiber.Handler {
		return middleware.Require(grpcClient, middleware.Params{Service: prefix, Entity: "chat", Action: action})
	}

	// Принадлежность конкретному чату (участник ли caller) проверяется уже
	// внутри service — здесь только базовое "есть ли вообще доступ к чатам".

	// permission chat:read
	router.Get("/stream", require("read"), sse.New(sse.Config{Handler: handler.Stream}))
	router.Get("", require("read"), handler.ListMyChats)
	router.Get("/:id", require("read"), handler.GetChat)
	router.Get("/:id/messages", require("read"), handler.ListMessages)
	router.Get("/:id/participants", require("read"), handler.ListParticipants)

	// permission chat:create
	router.Post("", require("create"), handler.CreateChat)
	router.Post("/entity/:entityType/:entityId", require("create"), handler.GetOrCreateEntityChat)
	router.Post("/:id/messages", require("create"), handler.SendMessage)
	router.Post("/:id/messages/file", require("create"), handler.SendFileMessage)
	router.Post("/:id/typing", require("create"), handler.Typing)
	router.Post("/:id/participants", require("create"), handler.AddParticipant)

	// permission chat:edit
	router.Put("/:id/name", require("edit"), handler.RenameChat)
	router.Put("/:id/read", require("edit"), handler.MarkRead)
	router.Put("/:id/mute", require("edit"), handler.SetMuted)

	// permission chat:delete
	router.Delete("/:id/messages/:messageId", require("delete"), handler.DeleteMessage)
	router.Delete("/:id/participants/:userId", require("delete"), handler.RemoveParticipant)
	router.Delete("/:id", require("delete"), handler.DeleteChat)
}
