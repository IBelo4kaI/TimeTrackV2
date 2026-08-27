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
	results := h.service.RunNow(c.RequestCtx())

	sent := 0
	for _, r := range results {
		if r.Notified {
			sent++
		}
	}

	return response.Success(c, fiber.Map{
		"message": "Проверка выполнена",
		"sent":    sent,
		"results": results,
	})
}
