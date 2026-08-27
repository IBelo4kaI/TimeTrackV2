package timesheetreminder

import (
	"timetrack/internal/response"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) Handler {
	return Handler{service: service}
}

// RunNow godoc
// POST /timesheet-reminder/run — ручной прогон проверки (см. Service.RunNow),
// не дожидаясь суточного тикера. Для проверки/на случай если нужно
// пересчитать раньше расписания.
func (h Handler) RunNow(c fiber.Ctx) error {
	sent := h.service.RunNow(c.RequestCtx())
	return response.Success(c, fiber.Map{
		"message": "Проверка выполнена",
		"sent":    sent,
	})
}
