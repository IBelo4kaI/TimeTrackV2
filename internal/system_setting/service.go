package systemsetting

import (
	"context"
	"database/sql"
	"github.com/gofiber/fiber/v3"
	"strings"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"unicode"
	"unicode/utf8"
)

type Service interface {
	UpdateSystemSettingValue(ctx context.Context, settingKey string, settingValue string) error
	GetSystemSettingByKey(ctx context.Context, settingKey string) (repo.SystemSetting, error)
}

type service struct {
	repo repo.Querier
}

func NewService(repo repo.Querier) Service {
	return &service{repo: repo}
}

// UpdateSystemSettingValue обновляет значение настройки по ключу
func (s *service) UpdateSystemSettingValue(ctx context.Context, settingKey string, settingValue string) error {
	if settingKey == "department_manager_role_id" {
		settingValue = strings.TrimSpace(settingValue)
		if utf8.RuneCountInString(settingValue) > 255 || strings.IndexFunc(settingValue, unicode.IsControl) >= 0 {
			return fiber.NewError(400, "ID роли должен содержать не более 255 символов без управляющих символов")
		}
	}
	params := repo.UpdateValueSystemSettingParams{
		SettingKey: settingKey,
		SettingValue: sql.NullString{
			String: settingValue,
			Valid:  settingValue != "",
		},
	}
	return s.repo.UpdateValueSystemSetting(ctx, params)
}

// GetSystemSettingByKey получает настройку по ключу
func (s *service) GetSystemSettingByKey(ctx context.Context, settingKey string) (repo.SystemSetting, error) {
	return s.repo.GetSystemSettingByKey(ctx, settingKey)
}
