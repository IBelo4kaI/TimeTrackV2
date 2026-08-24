package systemsetting

import (
	"database/sql"
	"errors"
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
		if errors.Is(err, sql.ErrNoRows) {
			// Настройку ещё ни разу не сохраняли — не ошибка, просто пусто.
			// UpdateSystemSettingValue теперь сам создаёт строку при первом
			// сохранении (см. UpdateValueSystemSetting), но до первого
			// сохранения запись законно может не существовать.
			return response.Success(c, fiber.Map{
				"settingKey":   settingKey,
				"settingValue": nil,
			})
		}
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, setting)
}
