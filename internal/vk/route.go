package vk

import (
	"context"
	"timetrack/internal/adapter/grpc"
	"timetrack/internal/middleware"

	"github.com/SevereCloud/vksdk/v3/callback"
	"github.com/SevereCloud/vksdk/v3/events"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
)

// Config — данные из настроек Callback API сообщества VK (см. .env_example
// VK_CONFIRMATION_STRING/VK_SECRET_KEY).
type Config struct {
	ConfirmationString string
	SecretKey          string
}

func SetupRoutes(f fiber.Router, service Service, grpcClient *grpc.Client, prefix string, cfg Config) {
	handler := NewHandler(service)
	router := f.Group("/vk")

	require := func(action string) fiber.Handler {
		return middleware.Require(grpcClient, middleware.Params{Service: prefix, Entity: "vk", Action: action})
	}

	// permission vk:*
	router.Post("/link-code", require("create"), handler.GenerateLinkCode)
	router.Delete("/link", require("delete"), handler.Unlink)
	router.Get("/status", require("read"), handler.Status)

	cb := callback.NewCallback()
	cb.ConfirmationKey = cfg.ConfirmationString
	cb.SecretKey = cfg.SecretKey

	cb.MessageNew(func(ctx context.Context, obj events.MessageNewObject) {
		service.HandleMessage(ctx, obj.Message.FromID, obj.Message.Text)
	})

	// Без middleware.Require — сюда стучится сам VK, не браузер с сессией;
	// подлинность запроса проверяет cb.HandleFunc по секретному ключу.
	router.Post("/callback", adaptor.HTTPHandlerFunc(cb.HandleFunc))
}
