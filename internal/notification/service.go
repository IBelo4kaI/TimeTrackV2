package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/vk"

	"github.com/google/uuid"
)

// Ключи в system_settings (каждый — JSON-массив user_id), отдельно для
// отпусков и больничных — настраиваются на странице "Настройки", не
// хардкожены. См. GetVacationAdminRecipients/GetSickLeaveAdminRecipients.
const (
	vacationAdminRecipientsSettingKey    = "notification_vacation_admin_user_ids"
	sickLeaveAdminRecipientsSettingKey   = "notification_sick_leave_admin_user_ids"
	vacationApprovedRecipientsSettingKey = "notification_vacation_approved_user_ids"
)

type Service interface {
	ListMine(ctx context.Context, userID string, limit, offset int32) ([]repo.Notification, error)
	CountUnread(ctx context.Context, userID string) (int64, error)
	MarkRead(ctx context.Context, id, userID string) error
	MarkAllRead(ctx context.Context, userID string) error
	// Delete/DeleteAll — насовсем, не просто пометка прочитанным.
	Delete(ctx context.Context, id, userID string) error
	DeleteAll(ctx context.Context, userID string) error
	// MarkReadByEntity — разом все накопленные уведомления по сущности
	// (например, чату) для пользователя, при прочтении самой сущности (см.
	// chat.Service.MarkRead). Best-effort в вызывающей стороне — как
	// CreateMany, ошибку только логируем.
	MarkReadByEntity(ctx context.Context, userID, entityType, entityID string) error

	// CreateMany — рассылка одного уведомления сразу нескольким пользователям
	// (например, всем админам о новой заявке). Best-effort: ошибки только
	// логирует, не возвращает — уведомление не должно ронять создание
	// заявки, ради которой шлётся.
	CreateMany(ctx context.Context, userIDs []string, title, message string, notifType repo.NotificationsType, entityType, entityID string)

	// SendManual — ручная рассылка от админа сотрудникам (свободный текст
	// или заранее заполненный на фронте из notification_template — сам бэк
	// про шаблоны не знает). В отличие от CreateMany это прямое действие
	// пользователя, а не побочный эффект другого действия — ошибку
	// возвращаем, а не просто логируем.
	SendManual(ctx context.Context, userIDs []string, title, message string) error

	// GetVacationAdminRecipients/GetSickLeaveAdminRecipients — раздельные
	// списки получателей. Пустой список, если настройка не задана —
	// вызывающая сторона просто никому не шлёт.
	GetVacationAdminRecipients(ctx context.Context) ([]string, error)
	GetSickLeaveAdminRecipients(ctx context.Context) ([]string, error)
	// GetVacationApprovedRecipients — отдельный список, не пересекается с
	// GetVacationAdminRecipients: тем шлют про НОВЫЕ заявки, этим — про уже
	// утверждённые (например, бухгалтерия).
	GetVacationApprovedRecipients(ctx context.Context) ([]string, error)

	// SSE
	Subscribe(userID string) chan Event
	Unsubscribe(userID string, ch chan Event)
}

type service struct {
	repo   repo.Querier
	hub    *Hub
	vk     vk.Service
	logger *slog.Logger
}

func NewService(r repo.Querier, hub *Hub, vkService vk.Service, logger *slog.Logger) Service {
	return &service{repo: r, hub: hub, vk: vkService, logger: logger}
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

func (s *service) Delete(ctx context.Context, id, userID string) error {
	if err := s.repo.DeleteNotification(ctx, repo.DeleteNotificationParams{ID: id, UserID: userID}); err != nil {
		return err
	}

	s.hub.SendToUser(userID, Event{Type: "notification_deleted", Data: map[string]string{"id": id}})

	return nil
}

func (s *service) DeleteAll(ctx context.Context, userID string) error {
	if err := s.repo.DeleteAllNotificationsByUser(ctx, userID); err != nil {
		return err
	}

	s.hub.SendToUser(userID, Event{Type: "notifications_cleared", Data: nil})

	return nil
}

func (s *service) MarkReadByEntity(ctx context.Context, userID, entityType, entityID string) error {
	if err := s.repo.MarkNotificationsReadByEntity(ctx, repo.MarkNotificationsReadByEntityParams{
		UserID:     userID,
		EntityType: sql.NullString{String: entityType, Valid: entityType != ""},
		EntityID:   sql.NullString{String: entityID, Valid: entityID != ""},
	}); err != nil {
		return err
	}

	// Сколько именно строк прочитано — не знаем (:exec не отдаёт rows
	// affected), поэтому шлём просто "по этой сущности у тебя теперь всё
	// прочитано" — фронт сам сверяет с тем, что у него загружено (см.
	// notificationCenter.js), лишний/нулевой эффект безопасен.
	s.hub.SendToUser(userID, Event{
		Type: "notifications_read",
		Data: map[string]string{"entityType": entityType, "entityId": entityID},
	})

	return nil
}

func (s *service) CreateMany(ctx context.Context, userIDs []string, title, message string, notifType repo.NotificationsType, entityType, entityID string) {
	typ := repo.NullNotificationsType{NotificationsType: notifType, Valid: notifType != ""}
	entType := sql.NullString{String: entityType, Valid: entityType != ""}
	entID := sql.NullString{String: entityID, Valid: entityID != ""}

	for _, userID := range userIDs {
		id := uuid.NewString()
		err := s.repo.CreateNotification(ctx, repo.CreateNotificationParams{
			ID:         id,
			UserID:     userID,
			Title:      title,
			Message:    message,
			Type:       typ,
			EntityType: entType,
			EntityID:   entID,
		})
		if err != nil {
			s.logger.Error("notification: create failed", "err", err, "userId", userID)
			continue
		}

		s.hub.SendToUser(userID, Event{
			Type: "notification_created",
			Data: repo.Notification{
				ID:         id,
				UserID:     userID,
				Title:      title,
				Message:    message,
				Type:       typ,
				IsRead:     sql.NullBool{Bool: false, Valid: true},
				EntityType: entType,
				EntityID:   entID,
				// .UTC() — та же логика, что и в 013_chat_timestamps_utc.sql:
				// created_at в БД теперь буквальный UTC (UTC_TIMESTAMP() в
				// CreateNotification), значение в SSE-пуше должно совпадать,
				// а не быть локальным временем процесса.
				CreatedAt: time.Now().UTC(),
			},
		})
	}
}

const manualEntityType = "admin_message"

func (s *service) SendManual(ctx context.Context, userIDs []string, title, message string) error {
	if len(userIDs) == 0 {
		return errors.New("выберите хотя бы одного получателя")
	}
	if title == "" {
		return errors.New("укажите заголовок уведомления")
	}

	s.CreateMany(ctx, userIDs, title, message, repo.NotificationsTypeInfo, manualEntityType, "")

	vkText := title
	if message != "" {
		vkText = title + ": " + message
	}
	s.vk.NotifyMany(ctx, userIDs, vkText, "")

	return nil
}

func (s *service) GetVacationAdminRecipients(ctx context.Context) ([]string, error) {
	return s.getRecipients(ctx, vacationAdminRecipientsSettingKey)
}

func (s *service) GetSickLeaveAdminRecipients(ctx context.Context) ([]string, error) {
	return s.getRecipients(ctx, sickLeaveAdminRecipientsSettingKey)
}

func (s *service) GetVacationApprovedRecipients(ctx context.Context) ([]string, error) {
	return s.getRecipients(ctx, vacationApprovedRecipientsSettingKey)
}

func (s *service) Subscribe(userID string) chan Event {
	return s.hub.Subscribe(userID)
}

func (s *service) Unsubscribe(userID string, ch chan Event) {
	s.hub.Unsubscribe(userID, ch)
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
