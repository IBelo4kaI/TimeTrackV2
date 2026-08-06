package systemsetting

import (
	"context"
	"database/sql"
	repo "timetrack/internal/adapter/mysql/sqlc"
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
