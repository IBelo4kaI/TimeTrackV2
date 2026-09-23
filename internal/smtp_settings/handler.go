package smtpsettings

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

type updateRequest struct {
	Host     string  `json:"host"`
	Port     int     `json:"port"`
	Username string  `json:"username"`
	From     string  `json:"from"`
	Password *string `json:"password"`
}

func (h *Handler) GetSettings(c fiber.Ctx) error {
	settings, err := h.service.GetPublicSettings(c.RequestCtx())
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}
	return response.Success(c, settings)
}

// UpdateSettings — пустая строка в password от фронта означает "не менять"
// (поле-заглушка вместо реального пароля в форме), не "очистить".
func (h *Handler) UpdateSettings(c fiber.Ctx) error {
	var req updateRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, err)
	}

	password := req.Password
	if password != nil && *password == "" {
		password = nil
	}

	if err := h.service.UpdateSettings(c.RequestCtx(), UpdateParams{
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		From:     req.From,
		Password: password,
	}); err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	settings, err := h.service.GetPublicSettings(c.RequestCtx())
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}
	return response.Success(c, settings)
}
