package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	repo "timetrack/internal/adapter/mysql/sqlc"
)

// Ключи в system_settings (каждый — JSON-массив user_id), отдельно для
// отпусков и больничных — настраиваются на странице "Настройки", не
// хардкожены. См. GetVacationAdminRecipients/GetSickLeaveAdminRecipients.
const (
	vacationAdminRecipientsSettingKey  = "notification_vacation_admin_user_ids"
	sickLeaveAdminRecipientsSettingKey = "notification_sick_leave_admin_user_ids"
)

type Service interface {
	ListMine(ctx context.Context, userID string, limit, offset int32) ([]repo.Notification, error)
	CountUnread(ctx context.Context, userID string) (int64, error)
	MarkRead(ctx context.Context, id, userID string) error
	MarkAllRead(ctx context.Context, userID string) error

	// CreateMany — рассылка одного уведомления сразу нескольким пользователям
	// (например, всем админам о новой заявке). Best-effort: ошибки только
	// логирует, не возвращает — уведомление не должно ронять создание
	// заявки, ради которой шлётся.
	CreateMany(ctx context.Context, userIDs []string, title, message string, notifType repo.NotificationsType, entityType, entityID string)

	// GetVacationAdminRecipients/GetSickLeaveAdminRecipients — раздельные
	// списки получателей. Пустой список, если настройка не задана —
	// вызывающая сторона просто никому не шлёт.
	GetVacationAdminRecipients(ctx context.Context) ([]string, error)
	GetSickLeaveAdminRecipients(ctx context.Context) ([]string, error)
}

type service struct {
	repo   repo.Querier
	logger *slog.Logger
}

func NewService(r repo.Querier, logger *slog.Logger) Service {
	return &service{repo: r, logger: logger}
}

func (s *service) ListMine(ctx context.Context, userID string, limit, offset int32) ([]repo.Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	return s.repo.ListNotificationsByUser(ctx, repo.ListNotificationsByUserParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
}

func (s *service) CountUnread(ctx context.Context, userID string) (int64, error) {
	return s.repo.CountUnreadNotifications(ctx, userID)
}

func (s *service) MarkRead(ctx context.Context, id, userID string) error {
	return s.repo.MarkNotificationRead(ctx, repo.MarkNotificationReadParams{ID: id, UserID: userID})
}

func (s *service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllNotificationsRead(ctx, userID)
}

func (s *service) CreateMany(ctx context.Context, userIDs []string, title, message string, notifType repo.NotificationsType, entityType, entityID string) {
	for _, userID := range userIDs {
		err := s.repo.CreateNotification(ctx, repo.CreateNotificationParams{
			UserID:     userID,
			Title:      title,
			Message:    message,
			Type:       repo.NullNotificationsType{NotificationsType: notifType, Valid: notifType != ""},
			EntityType: sql.NullString{String: entityType, Valid: entityType != ""},
			EntityID:   sql.NullString{String: entityID, Valid: entityID != ""},
		})
		if err != nil {
			s.logger.Error("notification: create failed", "err", err, "userId", userID)
		}
	}
}

func (s *service) GetVacationAdminRecipients(ctx context.Context) ([]string, error) {
	return s.getRecipients(ctx, vacationAdminRecipientsSettingKey)
}

func (s *service) GetSickLeaveAdminRecipients(ctx context.Context) ([]string, error) {
	return s.getRecipients(ctx, sickLeaveAdminRecipientsSettingKey)
}

func (s *service) getRecipients(ctx context.Context, settingKey string) ([]string, error) {
	setting, err := s.repo.GetSystemSettingByKey(ctx, settingKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if !setting.SettingValue.Valid || setting.SettingValue.String == "" {
		return nil, nil
	}

	var ids []string
	if err := json.Unmarshal([]byte(setting.SettingValue.String), &ids); err != nil {
		return nil, fmt.Errorf("parse %s: %w", settingKey, err)
	}
	return ids, nil
}
