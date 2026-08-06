package daytype

import (
	"timetrack/internal/adapter/grpc"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(fiber fiber.Router, service Service, grpc *grpc.Client, prefix string) {
	handler := NewHandler(service)
	router := fiber.Group("/daytypes")

	// permission для всех, нужна только авторизация
	router.Get("", handler.GetDayTypes)
}
