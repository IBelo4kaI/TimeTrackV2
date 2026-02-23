package service

import (
	"context"
	"database/sql"
	repo "timetrack/internal/adapter/mysql/sqlc"
)

type SystemSettingsService struct {
	repo repo.Querier
}

func NewSystemSettingsService(repo repo.Querier) *SystemSettingsService {
	return &SystemSettingsService{repo: repo}
}

// UpdateSystemSettingValue обновляет значение настройки по ключу
func (s *SystemSettingsService) UpdateSystemSettingValue(ctx context.Context, settingKey string, settingValue string) error {
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
func (s *SystemSettingsService) GetSystemSettingByKey(ctx context.Context, settingKey string) (repo.SystemSetting, error) {
	return s.repo.GetSystemSettingByKey(ctx, settingKey)
}
