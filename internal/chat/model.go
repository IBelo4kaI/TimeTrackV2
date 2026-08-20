package chat

import (
	"database/sql"
	repo "timetrack/internal/adapter/mysql/sqlc"
)

type CreateChatRequest struct {
	// "direct" | "group"
	Type           string   `json:"type"`
	Name           string   `json:"name"`
	ParticipantIDs []string `json:"participantIds"`
}

type SendMessageRequest struct {
	Body string `json:"body"`
	// Необязательная ссылка на сущность (например, заявку на отпуск) —
	// EntityType/EntityID должны быть заданы вместе или не заданы вовсе.
	// Title/Subtitle — готовый для отображения снимок с фронта (бэк
	// содержимое сущности не проверяет и не резолвит, см. миграцию 012).
	EntityType     string `json:"entityType"`
	EntityID       string `json:"entityId"`
	EntityTitle    string `json:"entityTitle"`
	EntitySubtitle string `json:"entitySubtitle"`
}

// EntityRef — распарсенная и провалидированная ссылка на сущность из
// SendMessageRequest, что удобнее гонять по service, чем 4 голые строки.
type EntityRef struct {
	Type     string
	ID       string
	Title    string
	Subtitle string
}

type MarkReadRequest struct {
	MessageID uint64 `json:"messageId"`
}

type AddParticipantRequest struct {
	UserID string `json:"userId"`
	// "member" | "admin", по умолчанию member (см. service.go)
	Role string `json:"role"`
}

// MessageAttachment — файл, прикреплённый к сообщению (files + file_entity_refs
// с entity_type='chat_message', entity_id=<id сообщения>). Урезанный набор
// полей — того, что нужно фронту для превью/скачивания; сам файл отдаётся
// через уже существующий GET /v1/files/open/:id.
type MessageAttachment struct {
	ID           string `json:"id"`
	OriginalName string `json:"originalName"`
	MimeType     string `json:"mimeType"`
	FileType     string `json:"fileType"`
	SizeBytes    int64  `json:"sizeBytes"`
}

// ChatMessageDTO — сообщение + его вложения. Attachments всегда непустой
// срез (даже если пустой), чтобы фронту не пришлось разбирать null.
type ChatMessageDTO struct {
	repo.ChatMessage
	Attachments []MessageAttachment `json:"attachments"`
}

// ChatWithMeta — чат + то, что относится к КОНКРЕТНОМУ пользователю,
// который его запросил (его роль, курсор прочитанного, счётчик непрочитанных).
// Отдаём это, а не голую repo.Chat, чтобы фронту не пришлось для каждого
// чата в списке делать отдельный запрос за непрочитанными.
type ChatWithMeta struct {
	repo.Chat
	Role              repo.ChatParticipantsRole `json:"role"`
	LastReadMessageID *uint64                   `json:"lastReadMessageId"`
	UnreadCount       int64                     `json:"unreadCount"`
}

func nullInt64ToPtr(v sql.NullInt64) *uint64 {
	if !v.Valid {
		return nil
	}
	u := uint64(v.Int64)
	return &u
}
