package timesheetreminder

import (
	"timetrack/internal/adapter/grpc"
	"timetrack/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(f fiber.Router, service *Service, grpcClient *grpc.Client, prefix string) {
	handler := NewHandler(service)

	// permission system_settings:edit — та же админская планка, что и у
	// управления получателями уведомлений; это операционный триггер
	// фонового job'а, не личное действие сотрудника, поэтому не
	// notification:edit (тот выдан всем — про свои уведомления).
	f.Post("/timesheet-reminder/run",
		middleware.Require(grpcClient, middleware.Params{Service: prefix, Entity: "system_settings", Action: "edit"}),
		handler.RunNow)
}
