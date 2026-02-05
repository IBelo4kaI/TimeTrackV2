package handler

import (
	"net/http"
	"timetrack/internal/response"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v2"
)

type DayTypeHandler struct {
	service service.DayTypeService
}

func NewDayTypeHandler(service service.DayTypeService) DayTypeHandler {
	return DayTypeHandler{service: service}
}

func (h DayTypeHandler) GetDayTypes(c *fiber.Ctx) error {
	types, err := h.service.GetDayTypes(c.Context())
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}
	return response.Success(c, types)
}
