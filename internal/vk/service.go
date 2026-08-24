package vk

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"math/big"
	"strings"
	"sync"
	"time"

	repo "timetrack/internal/adapter/mysql/sqlc"

	"github.com/SevereCloud/vksdk/v3/api"
	"github.com/SevereCloud/vksdk/v3/object"
)

// LinkCodeTTL — сколько код привязки действителен после генерации.
const LinkCodeTTL = 10 * time.Minute

// без 0/O/1/I — легче набрать вручную, отправляя боту сообщением.
const linkCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

type Service interface {
	// GenerateLinkCode — код, который сотрудник присылает боту сообщением,
	// чтобы привязать VK-аккаунт (см. HandleMessage). Хранится в памяти —
	// код короткоживущий, переживать рестарт ему не нужно.
	GenerateLinkCode(userID string) string
	Unlink(ctx context.Context, userID string) error
	IsLinked(ctx context.Context, userID string) (bool, error)

	// HandleMessage — входящее сообщение от пользователя VK (Callback API
	// message_new). Единственное, что понимаем сейчас — код привязки.
	HandleMessage(ctx context.Context, vkUserID int, text string)

	// Notify — тихо ничего не делает, если аккаунт не привязан (VK для
	// уведомлений опционален, а не обязателен).
	Notify(ctx context.Context, userID, text, url string)
	// NotifyMany — то же самое пачкой, exclude — кого пропустить
	// (например, отправителя сообщения в чате).
	NotifyMany(ctx context.Context, userIDs []string, text, url string, exclude ...string)
}

type pendingCode struct {
	userID    string
	expiresAt time.Time
}

type service struct {
	repo   repo.Querier
	vk     *api.VK
	logger *slog.Logger

	mu    sync.Mutex
	codes map[string]pendingCode
}

func NewService(r repo.Querier, groupToken string, logger *slog.Logger) Service {
	return &service{
		repo:   r,
		vk:     api.NewVK(groupToken),
		logger: logger,
		codes:  make(map[string]pendingCode),
	}
}

func (s *service) GenerateLinkCode(userID string) string {
	code := randomCode()

	s.mu.Lock()
	s.codes[code] = pendingCode{userID: userID, expiresAt: time.Now().Add(LinkCodeTTL)}
	s.mu.Unlock()

	return code
}

func (s *service) Unlink(ctx context.Context, userID string) error {
	return s.repo.UnlinkUserVK(ctx, userID)
}

func (s *service) IsLinked(ctx context.Context, userID string) (bool, error) {
	_, err := s.repo.GetVKIDByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *service) HandleMessage(ctx context.Context, vkUserID int, text string) {
	code := strings.ToUpper(strings.TrimSpace(text))

	s.mu.Lock()
	pending, ok := s.codes[code]
	if ok {
		delete(s.codes, code)
	}
	s.mu.Unlock()

	if !ok || time.Now().After(pending.expiresAt) {
		s.send(vkUserID, "Код не найден или истёк — сгенерируйте новый в приложении.", "")
		return
	}

	if err := s.repo.LinkUserVK(ctx, repo.LinkUserVKParams{
		UserID:   pending.userID,
		VkUserID: int64(vkUserID),
	}); err != nil {
		s.logger.Error("vk: link account failed", "err", err)
		s.send(vkUserID, "Не удалось привязать аккаунт, попробуйте ещё раз.", "")
		return
	}

	s.send(vkUserID, "Готово — уведомления теперь будут дублироваться сюда.", "")
}

func (s *service) Notify(ctx context.Context, userID, text, url string) {
	vkID, err := s.repo.GetVKIDByUser(ctx, userID)
	if err != nil {
		return // не привязан либо БД недоступна — не критично, это дублирующий канал
	}
	s.send(int(vkID), text, url)
}

func (s *service) NotifyMany(ctx context.Context, userIDs []string, text, url string, exclude ...string) {
	excluded := make(map[string]struct{}, len(exclude))
	for _, id := range exclude {
		excluded[id] = struct{}{}
	}

	ids := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		if _, skip := excluded[id]; !skip {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return
	}

	links, err := s.repo.ListVKIDsByUsers(ctx, ids)
	if err != nil {
		return
	}

	for _, l := range links {
		s.send(int(l.VkUserID), text, url)
	}
}

// send — url (если есть) вложен как inline-кнопка "Перейти в чат", а не
// вставлен в текст сырой строкой: так короче и не занимает отдельную строку
// сообщения.
func (s *service) send(vkUserID int, message, url string) {
	params := api.Params{
		"user_id":   vkUserID,
		"message":   message,
		"random_id": time.Now().UnixNano(),
	}

	if url != "" {
		// Не через AddOpenLinkButton — она маршалит payload=nil в строку
		// "null" (VK: "invalid payload"). Кнопке open_link payload не
		// нужен вовсе, оставляем поле пустым (omitempty уберёт его из JSON).
		kb := object.NewMessagesKeyboardInline()
		kb.AddRow()
		kb.Buttons[0] = append(kb.Buttons[0], object.MessagesKeyboardButton{
			Action: object.MessagesKeyboardButtonAction{
				Type:  object.ButtonOpenLink,
				Label: "Перейти в чат",
				Link:  url,
			},
		})
		if raw, err := json.Marshal(kb); err == nil {
			params["keyboard"] = string(raw)
		}
	}

	if _, err := s.vk.MessagesSend(params); err != nil {
		s.logger.Error("vk: send message failed", "err", err, "vkUserId", vkUserID)
	}
}

func randomCode() string {
	b := make([]byte, 6)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(linkCodeAlphabet))))
		b[i] = linkCodeAlphabet[n.Int64()]
	}
	return string(b)
}
