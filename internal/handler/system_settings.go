package handler

import (
	"net/http"
	"timetrack/internal/response"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v2"
)

type SystemSettingsHandler struct {
	service *service.SystemSettingsService
}

func NewSystemSettingsHandler(service *service.SystemSettingsService) *SystemSettingsHandler {
	return &SystemSettingsHandler{service: service}
}

// UpdateSystemSettingValue обновляет значение настройки
func (h *SystemSettingsHandler) UpdateSystemSettingValue(c *fiber.Ctx) error {
	type request struct {
		SettingKey   string `json:"settingKey"`
		SettingValue string `json:"settingValue"`
	}

	var req request
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, err)
	}

	if req.SettingKey == "" {
		return fiber.NewError(http.StatusBadRequest, "setting_key is required")
	}

	err := h.service.UpdateSystemSettingValue(c.Context(), req.SettingKey, req.SettingValue)
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
func (h *SystemSettingsHandler) GetSystemSettingByKey(c *fiber.Ctx) error {
	settingKey := c.Params("key")
	if settingKey == "" {
		return fiber.NewError(http.StatusBadRequest, "setting key is required")
	}

	setting, err := h.service.GetSystemSettingByKey(c.Context(), settingKey)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, setting)
}
