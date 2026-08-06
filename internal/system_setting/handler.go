package systemsetting

import (
	"net/http"
	"timetrack/internal/response"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// UpdateSystemSettingValue обновляет значение настройки
func (h *Handler) UpdateSystemSettingValue(c fiber.Ctx) error {
	type request struct {
		SettingKey   string `json:"settingKey"`
		SettingValue string `json:"settingValue"`
	}

	var req request
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, err)
	}

	if req.SettingKey == "" {
		return fiber.NewError(http.StatusBadRequest, "setting_key is required")
	}

	err := h.service.UpdateSystemSettingValue(c.RequestCtx(), req.SettingKey, req.SettingValue)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, fiber.Map{
		"message":      "Setting updated successfully",
		"settingKey":   req.SettingKey,
		"settingValue": req.SettingValue,
	})
}

// GetSystemSettingByKey получает настройку по ключу
func (h *Handler) GetSystemSettingByKey(c fiber.Ctx) error {
	settingKey := c.Params("key")
	if settingKey == "" {
		return fiber.NewError(http.StatusBadRequest, "setting key is required")
	}

	setting, err := h.service.GetSystemSettingByKey(c.RequestCtx(), settingKey)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, setting)
}
