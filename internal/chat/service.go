package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"mime/multipart"
	"strconv"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
	fileservice "timetrack/internal/service"
	"timetrack/internal/vk"

	"github.com/google/uuid"
)

var (
	ErrChatNotFound    = errors.New("чат не найден")
	ErrMessageNotFound = errors.New("сообщение не найдено")
	ErrNotParticipant  = errors.New("вы не участник этого чата")
	ErrNotOwnMessage   = errors.New("удалить можно только своё сообщение")
	ErrNotAllowed      = errors.New("удалить групповой чат может только его создатель или админ чата")
	ErrBadChatType     = errors.New("type должен быть 'direct' или 'group'")
	ErrNoParticipants  = errors.New("нужен хотя бы один участник, кроме себя")
	ErrEmptyBody       = errors.New("сообщение не может быть пустым")
	ErrDirectTwoUsers  = errors.New("в личном чате ровно два участника")
	// ErrBadEntityRef — исторически "entityType/entityId заданы не вместе",
	// теперь ещё и "неизвестный/неподдерживаемый entityType" (см.
	// ResolveEntityRefOwner) — в обоих случаях это 400, ссылка невалидна.
	ErrBadEntityRef       = errors.New("entityType и entityId должны быть заданы вместе")
	ErrEntityRefNotFound  = errors.New("сущность, на которую ссылается сообщение, не найдена")
	ErrEntityRefForbidden = errors.New("нет доступа, чтобы сослаться на эту заявку")
)

const defaultMessagesPageSize = 50

type Service interface {
	// Чаты
	CreateChat(ctx context.Context, callerUserID string, req CreateChatRequest) (ChatWithMeta, error)
	GetOrCreateEntityChat(ctx context.Context, entityType, entityID, callerUserID string, otherUserIDs []string) (ChatWithMeta, error)
	GetChat(ctx context.Context, chatID, callerUserID string) (ChatWithMeta, error)
	ListMyChats(ctx context.Context, callerUserID string) ([]ChatWithMeta, error)
	RenameChat(ctx context.Context, chatID, callerUserID, name string) error
	SetMuted(ctx context.Context, chatID, callerUserID string, muted bool) error
	DeleteChat(ctx context.Context, chatID, callerUserID string) error

	// Сообщения
	ListMessages(ctx context.Context, chatID, callerUserID string, beforeID *uint64, limit int32) ([]ChatMessageDTO, error)
	SendMessage(ctx context.Context, chatID, callerUserID, body string, ref *EntityRef) (ChatMessageDTO, error)
	SendFileMessage(ctx context.Context, chatID, callerUserID, caption string, files []*multipart.FileHeader) (ChatMessageDTO, error)
	// ResolveEntityRefOwner — userID владельца сущности, на которую
	// ссылается сообщение (сейчас только "vacation"), нужен хендлеру для
	// авторизации ссылки (см. handler.go checkEntityRefAccess): на СВОЮ
	// заявку сослаться может любой, на ЧУЖУЮ — только с отдельным
	// <entityType>.all:link. Неизвестный entityType — ErrBadEntityRef.
	ResolveEntityRefOwner(ctx context.Context, entityType, entityID string) (string, error)
	DeleteMessage(ctx context.Context, messageID uint64, callerUserID string) error
	MarkRead(ctx context.Context, chatID, callerUserID string, messageID uint64) error
	Typing(ctx context.Context, chatID, callerUserID string) error

	// Участники
	ListParticipants(ctx context.Context, chatID, callerUserID string) ([]repo.ChatParticipant, error)
	AddParticipant(ctx context.Context, chatID, callerUserID, newUserID, role string) error
	RemoveParticipant(ctx context.Context, chatID, callerUserID, targetUserID string) error

	// SSE
	Subscribe(userID string) chan Event
	Unsubscribe(userID string, ch chan Event)
}

// entityTypeChatMessage — entity_type для файлов, прикреплённых к сообщению
// (file_entity_refs.entity_id = id сообщения, как строка). Отдельных таблиц
// для вложений не заводим — переиспользуем ту же полиморфную схему files/
// file_entity_refs, что уже используется для файлов отпусков/больничных.
const entityTypeChatMessage = "chat_message"

type service struct {
	repo        repo.Querier
	hub         *Hub
	fileService *fileservice.FileService
	vk          vk.Service
	frontendURL string
}

func NewService(r repo.Querier, hub *Hub, fileService *fileservice.FileService, vkService vk.Service, frontendURL string) Service {
	return &service{repo: r, hub: hub, fileService: fileService, vk: vkService, frontendURL: frontendURL}
}

// ============================================
// Чаты
// ============================================

func (s *service) CreateChat(ctx context.Context, callerUserID string, req CreateChatRequest) (ChatWithMeta, error) {
	participantIDs, err := normalizeParticipants(req.Type, callerUserID, req.ParticipantIDs)
	if err != nil {
		return ChatWithMeta{}, err
	}

	chatID := uuid.NewString()
	chatType := repo.ChatsType(req.Type)

	if err := s.repo.CreateChat(ctx, repo.CreateChatParams{
		ID:              chatID,
		Type:            chatType,
		Name:            sql.NullString{String: req.Name, Valid: req.Name != ""},
		CreatedByUserID: callerUserID,
	}); err != nil {
		return ChatWithMeta{}, fmt.Errorf("create chat: %w", err)
	}

	for _, uid := range participantIDs {
		role := repo.ChatParticipantsRoleMember
		if uid == callerUserID {
			role = repo.ChatParticipantsRoleAdmin
		}
		if err := s.repo.AddChatParticipant(ctx, repo.AddChatParticipantParams{
			ChatID: chatID,
			UserID: uid,
			Role:   role,
		}); err != nil {
			return ChatWithMeta{}, fmt.Errorf("add participant %s: %w", uid, err)
		}
	}

	// Остальным участникам чат создал не они — без этого события их список
	// чатов не узнает о новом чате, пока они сами не перезайдут на страницу.
	s.broadcastToParticipants(ctx, chatID, Event{
		Type: EventChatCreated,
		Data: map[string]any{"chatId": chatID},
	}, callerUserID)
	s.notifyVK(ctx, chatID, "У вас новый чат", callerUserID)

	return s.GetChat(ctx, chatID, callerUserID)
}

// GetOrCreateEntityChat — «обсуждение заявки»: находит уже существующий чат,
// привязанный к сущности (entity_type/entity_id), либо создаёт новый и сразу
// добавляет туда caller'а и otherUserIDs (например, сотрудника и админа,
// которые обсуждают конкретный отпуск). Идемпотентно — безопасно дёргать
// повторно с тех же параметров.
func (s *service) GetOrCreateEntityChat(ctx context.Context, entityType, entityID, callerUserID string, otherUserIDs []string) (ChatWithMeta, error) {
	existing, err := s.repo.GetChatByEntity(ctx, repo.GetChatByEntityParams{
		EntityType: sql.NullString{String: entityType, Valid: true},
		EntityID:   sql.NullString{String: entityID, Valid: true},
	})
	if err == nil {
		// Чат уже есть — на случай, если caller ещё не был участником
		// (например, назначен новый ответственный), добавляем его.
		if _, pErr := s.repo.GetChatParticipant(ctx, repo.GetChatParticipantParams{
			ChatID: existing.ID,
			UserID: callerUserID,
		}); pErr != nil {
			if !errors.Is(pErr, sql.ErrNoRows) {
				return ChatWithMeta{}, pErr
			}
			if aErr := s.repo.AddChatParticipant(ctx, repo.AddChatParticipantParams{
				ChatID: existing.ID,
				UserID: callerUserID,
				Role:   repo.ChatParticipantsRoleMember,
			}); aErr != nil {
				return ChatWithMeta{}, fmt.Errorf("add caller to entity chat: %w", aErr)
			}
			// Остальным участникам — что в их чате появился ещё один человек
			// (список чатов у них не поменялся, но состав — да; фронт по
			// этому событию просто перечитает список/участников).
			s.broadcastToParticipants(ctx, existing.ID, Event{
				Type: EventChatCreated,
				Data: map[string]any{"chatId": existing.ID},
			}, callerUserID)
			s.notifyVK(ctx, existing.ID, "У вас новый чат", callerUserID)
		}
		return s.GetChat(ctx, existing.ID, callerUserID)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return ChatWithMeta{}, err
	}

	participantIDs := append([]string{callerUserID}, otherUserIDs...)
	participantIDs = dedupe(participantIDs)

	chatID := uuid.NewString()
	if err := s.repo.CreateChat(ctx, repo.CreateChatParams{
		ID:              chatID,
		Type:            repo.ChatsTypeGroup,
		EntityType:      sql.NullString{String: entityType, Valid: true},
		EntityID:        sql.NullString{String: entityID, Valid: true},
		CreatedByUserID: callerUserID,
	}); err != nil {
		return ChatWithMeta{}, fmt.Errorf("create entity chat: %w", err)
	}

	for _, uid := range participantIDs {
		if err := s.repo.AddChatParticipant(ctx, repo.AddChatParticipantParams{
			ChatID: chatID,
			UserID: uid,
			Role:   repo.ChatParticipantsRoleMember,
		}); err != nil {
			return ChatWithMeta{}, fmt.Errorf("add participant %s: %w", uid, err)
		}
	}

	s.broadcastToParticipants(ctx, chatID, Event{
		Type: EventChatCreated,
		Data: map[string]any{"chatId": chatID},
	}, callerUserID)
	s.notifyVK(ctx, chatID, "У вас новый чат", callerUserID)

	return s.GetChat(ctx, chatID, callerUserID)
}

func (s *service) GetChat(ctx context.Context, chatID, callerUserID string) (ChatWithMeta, error) {
	participant, err := s.ensureParticipant(ctx, chatID, callerUserID)
	if err != nil {
		return ChatWithMeta{}, err
	}

	c, err := s.repo.GetChatByID(ctx, chatID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ChatWithMeta{}, ErrChatNotFound
		}
		return ChatWithMeta{}, err
	}

	unread, err := s.repo.CountUnreadChatMessages(ctx, repo.CountUnreadChatMessagesParams{
		ChatID: chatID,
		UserID: callerUserID,
	})
	if err != nil {
		return ChatWithMeta{}, err
	}

	return ChatWithMeta{
		Chat:              c,
		Role:              participant.Role,
		LastReadMessageID: nullInt64ToPtr(participant.LastReadMessageID),
		UnreadCount:       unread,
		Muted:             participant.Muted,
	}, nil
}

func (s *service) ListMyChats(ctx context.Context, callerUserID string) ([]ChatWithMeta, error) {
	rows, err := s.repo.ListChatsByUser(ctx, callerUserID)
	if err != nil {
		return nil, err
	}

	result := make([]ChatWithMeta, 0, len(rows))
	for _, r := range rows {
		unread, err := s.repo.CountUnreadChatMessages(ctx, repo.CountUnreadChatMessagesParams{
			ChatID: r.ID,
			UserID: callerUserID,
		})
		if err != nil {
			return nil, err
		}

		result = append(result, ChatWithMeta{
			Chat: repo.Chat{
				ID:              r.ID,
				Type:            r.Type,
				Name:            r.Name,
				EntityType:      r.EntityType,
				EntityID:        r.EntityID,
				CreatedByUserID: r.CreatedByUserID,
				LastMessageAt:   r.LastMessageAt,
				CreatedAt:       r.CreatedAt,
				UpdatedAt:       r.UpdatedAt,
			},
			Role:              r.Role,
			LastReadMessageID: nullInt64ToPtr(r.LastReadMessageID),
			UnreadCount:       unread,
			Muted:             r.Muted,
		})
	}

	return result, nil
}

// SetMuted включает/выключает уведомления по чату лично для callerUserID —
// не влияет на остальных участников (см. миграцию 014_chat_mute.sql).
func (s *service) SetMuted(ctx context.Context, chatID, callerUserID string, muted bool) error {
	if _, err := s.ensureParticipant(ctx, chatID, callerUserID); err != nil {
		return err
	}
	return s.repo.SetChatParticipantMuted(ctx, repo.SetChatParticipantMutedParams{
		Muted:  muted,
		ChatID: chatID,
		UserID: callerUserID,
	})
}

func (s *service) RenameChat(ctx context.Context, chatID, callerUserID, name string) error {
	if _, err := s.ensureParticipant(ctx, chatID, callerUserID); err != nil {
		return err
	}
	return s.repo.UpdateChatName(ctx, repo.UpdateChatNameParams{
		Name: sql.NullString{String: name, Valid: name != ""},
		ID:   chatID,
	})
}

// DeleteChat удаляет чат целиком (участников и сообщения — каскадом, см.
// миграцию). Личный чат может удалить любой из двух участников (переписка
// вдвоём без второй стороны смысла не имеет). Групповой — только тот, кто
// его создал, либо участник с ролью admin; остальные выходят через
// RemoveParticipant (self), а не удаляют чат целиком у всех.
func (s *service) DeleteChat(ctx context.Context, chatID, callerUserID string) error {
	participant, err := s.ensureParticipant(ctx, chatID, callerUserID)
	if err != nil {
		return err
	}

	c, err := s.repo.GetChatByID(ctx, chatID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrChatNotFound
		}
		return err
	}

	if c.Type == repo.ChatsTypeGroup &&
		c.CreatedByUserID != callerUserID &&
		participant.Role != repo.ChatParticipantsRoleAdmin {
		return ErrNotAllowed
	}

	// Список участников — читаем ДО удаления (после каскада строк уже не будет),
	// чтобы разослать оставшимся, что чат исчез, и фронт убрал его из списка.
	participants, err := s.repo.ListChatParticipants(ctx, chatID)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteChat(ctx, chatID); err != nil {
		return fmt.Errorf("delete chat: %w", err)
	}

	for _, p := range participants {
		if p.UserID == callerUserID {
			continue
		}
		s.hub.SendToUser(p.UserID, Event{
			Type: EventChatDeleted,
			Data: map[string]any{"chatId": chatID},
		})
	}

	return nil
}

// ============================================
// Сообщения
// ============================================

func (s *service) ListMessages(ctx context.Context, chatID, callerUserID string, beforeID *uint64, limit int32) ([]ChatMessageDTO, error) {
	if _, err := s.ensureParticipant(ctx, chatID, callerUserID); err != nil {
		return nil, err
	}

	if limit <= 0 || limit > 200 {
		limit = defaultMessagesPageSize
	}

	before := sql.NullInt64{}
	if beforeID != nil {
		before = sql.NullInt64{Int64: int64(*beforeID), Valid: true}
	}

	messages, err := s.repo.ListChatMessages(ctx, repo.ListChatMessagesParams{
		ChatID:   chatID,
		BeforeID: before,
		Limit:    limit,
	})
	if err != nil {
		return nil, err
	}

	ids := make([]uint64, 0, len(messages))
	for _, m := range messages {
		ids = append(ids, m.ID)
	}
	attachments, err := s.attachmentsForMessages(ctx, ids)
	if err != nil {
		return nil, err
	}

	result := make([]ChatMessageDTO, 0, len(messages))
	for _, m := range messages {
		result = append(result, ChatMessageDTO{ChatMessage: m, Attachments: attachments[m.ID]})
	}
	return result, nil
}

func (s *service) SendMessage(ctx context.Context, chatID, callerUserID, body string, ref *EntityRef) (ChatMessageDTO, error) {
	if _, err := s.ensureParticipant(ctx, chatID, callerUserID); err != nil {
		return ChatMessageDTO{}, err
	}
	if body == "" && ref == nil {
		return ChatMessageDTO{}, ErrEmptyBody
	}

	message, err := s.createMessageRow(ctx, chatID, callerUserID, body, ref)
	if err != nil {
		return ChatMessageDTO{}, err
	}

	dto := ChatMessageDTO{ChatMessage: message, Attachments: []MessageAttachment{}}
	s.broadcastToParticipants(ctx, chatID, Event{Type: EventMessageCreated, Data: dto})
	s.notifyVK(ctx, chatID, vkMessagePreview(body), callerUserID)

	return dto, nil
}

// ResolveEntityRefOwner — см. комментарий в интерфейсе Service. Единственный
// сейчас поддерживаемый entityType — "vacation"; расширять на sick_leave и
// т.п. — добавлением case сюда, никаких других изменений не потребуется
// (хендлер уже проверяет <entityType>.all:link дженерик).
func (s *service) ResolveEntityRefOwner(ctx context.Context, entityType, entityID string) (string, error) {
	switch entityType {
	case "vacation":
		v, err := s.repo.GetVacationByID(ctx, entityID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", ErrEntityRefNotFound
			}
			return "", err
		}
		return v.UserID, nil
	default:
		return "", ErrBadEntityRef
	}
}

// maxAttachmentsPerMessage — защита от одного запроса с сотней файлов;
// разумного продуктового ограничения (в т.ч. на фронте) для чата достаточно.
const maxAttachmentsPerMessage = 10

// SendFileMessage — сообщение с одним или несколькими вложениями. Каждый
// файл привязывается к message.id через file_entity_refs
// (entity_type=chat_message), поэтому сначала создаётся сама строка
// сообщения (тело — необязательная подпись, может быть пустым), а уже потом
// по очереди грузятся файлы на её id. Если какой-то файл не загрузился —
// возвращаем ошибку сразу; сообщение и уже успевшие загрузиться до него
// вложения при этом остаются (не роллбэчим предыдущие — редкий edge case,
// не стоит городить ради него распределённую транзакцию по нескольким
// независимым upload'ам).
func (s *service) SendFileMessage(ctx context.Context, chatID, callerUserID, caption string, files []*multipart.FileHeader) (ChatMessageDTO, error) {
	if _, err := s.ensureParticipant(ctx, chatID, callerUserID); err != nil {
		return ChatMessageDTO{}, err
	}
	if len(files) == 0 {
		return ChatMessageDTO{}, errors.New("файл не найден в запросе")
	}
	if len(files) > maxAttachmentsPerMessage {
		return ChatMessageDTO{}, fmt.Errorf("не более %d файлов за раз", maxAttachmentsPerMessage)
	}

	message, err := s.createMessageRow(ctx, chatID, callerUserID, caption, nil)
	if err != nil {
		return ChatMessageDTO{}, err
	}

	attachments := make([]MessageAttachment, 0, len(files))
	for _, file := range files {
		uploaded, err := s.fileService.Upload(ctx, fileservice.UploadFileParams{
			File:       file,
			EntityType: entityTypeChatMessage,
			EntityID:   strconv.FormatUint(message.ID, 10),
			UploaderID: callerUserID,
		})
		if err != nil {
			return ChatMessageDTO{}, fmt.Errorf("upload attachment %q: %w", file.Filename, err)
		}
		attachments = append(attachments, MessageAttachment{
			ID:           uploaded.ID,
			OriginalName: uploaded.OriginalName,
			MimeType:     uploaded.MimeType,
			FileType:     uploaded.FileType,
			SizeBytes:    uploaded.SizeBytes,
		})
	}

	dto := ChatMessageDTO{ChatMessage: message, Attachments: attachments}
	s.broadcastToParticipants(ctx, chatID, Event{Type: EventMessageCreated, Data: dto})
	s.notifyVK(ctx, chatID, vkMessagePreview(caption), callerUserID)

	return dto, nil
}

// createMessageRow — общая часть SendMessage/SendFileMessage: вставка строки
// сообщения, чтение её обратно и обновление chats.last_message_at. ref может
// быть nil (обычное сообщение без ссылки на сущность).
func (s *service) createMessageRow(ctx context.Context, chatID, callerUserID, body string, ref *EntityRef) (repo.ChatMessage, error) {
	params := repo.CreateChatMessageParams{
		ChatID:       chatID,
		SenderUserID: callerUserID,
		Body:         body,
	}
	if ref != nil {
		params.EntityType = sql.NullString{String: ref.Type, Valid: true}
		params.EntityID = sql.NullString{String: ref.ID, Valid: true}
		params.EntityTitle = sql.NullString{String: ref.Title, Valid: ref.Title != ""}
		params.EntitySubtitle = sql.NullString{String: ref.Subtitle, Valid: ref.Subtitle != ""}
	}

	res, err := s.repo.CreateChatMessage(ctx, params)
	if err != nil {
		return repo.ChatMessage{}, fmt.Errorf("create chat message: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return repo.ChatMessage{}, err
	}

	message, err := s.repo.GetChatMessageByID(ctx, uint64(id))
	if err != nil {
		return repo.ChatMessage{}, err
	}

	now := sql.NullTime{Time: time.Now(), Valid: true}
	if err := s.repo.TouchChatLastMessage(ctx, repo.TouchChatLastMessageParams{
		LastMessageAt: now,
		ID:            chatID,
	}); err != nil {
		return repo.ChatMessage{}, err
	}

	return message, nil
}

func (s *service) DeleteMessage(ctx context.Context, messageID uint64, callerUserID string) error {
	message, err := s.repo.GetChatMessageByID(ctx, messageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrMessageNotFound
		}
		return err
	}

	if _, err := s.ensureParticipant(ctx, message.ChatID, callerUserID); err != nil {
		return err
	}
	if message.SenderUserID != callerUserID {
		return ErrNotOwnMessage
	}

	if err := s.repo.SoftDeleteChatMessage(ctx, messageID); err != nil {
		return err
	}

	s.broadcastToParticipants(ctx, message.ChatID, Event{
		Type: EventMessageDeleted,
		Data: map[string]any{"chatId": message.ChatID, "messageId": messageID},
	})

	return nil
}

func (s *service) MarkRead(ctx context.Context, chatID, callerUserID string, messageID uint64) error {
	if _, err := s.ensureParticipant(ctx, chatID, callerUserID); err != nil {
		return err
	}

	if err := s.repo.MarkChatRead(ctx, repo.MarkChatReadParams{
		LastReadMessageID: sql.NullInt64{Int64: int64(messageID), Valid: true},
		ChatID:            chatID,
		UserID:            callerUserID,
	}); err != nil {
		return err
	}

	s.broadcastToParticipants(ctx, chatID, Event{
		Type: EventReadReceipt,
		Data: map[string]any{"chatId": chatID, "userId": callerUserID, "messageId": messageID},
	})

	return nil
}

// Typing — эфемерный сигнал "печатает", ничего не пишет в БД.
func (s *service) Typing(ctx context.Context, chatID, callerUserID string) error {
	if _, err := s.ensureParticipant(ctx, chatID, callerUserID); err != nil {
		return err
	}

	s.broadcastToParticipants(ctx, chatID, Event{
		Type: EventTyping,
		Data: map[string]any{"chatId": chatID, "userId": callerUserID},
	}, callerUserID) // самому себе печатание показывать не нужно

	return nil
}

// ============================================
// Участники
// ============================================

func (s *service) ListParticipants(ctx context.Context, chatID, callerUserID string) ([]repo.ChatParticipant, error) {
	if _, err := s.ensureParticipant(ctx, chatID, callerUserID); err != nil {
		return nil, err
	}
	return s.repo.ListChatParticipants(ctx, chatID)
}

func (s *service) AddParticipant(ctx context.Context, chatID, callerUserID, newUserID, role string) error {
	if _, err := s.ensureParticipant(ctx, chatID, callerUserID); err != nil {
		return err
	}

	r := repo.ChatParticipantsRoleMember
	if role == string(repo.ChatParticipantsRoleAdmin) {
		r = repo.ChatParticipantsRoleAdmin
	}

	if err := s.repo.AddChatParticipant(ctx, repo.AddChatParticipantParams{
		ChatID: chatID,
		UserID: newUserID,
		Role:   r,
	}); err != nil {
		return err
	}

	// Новому участнику — иначе у него чат не появится в списке, пока сам не
	// перезайдёт на страницу чатов.
	s.hub.SendToUser(newUserID, Event{
		Type: EventChatCreated,
		Data: map[string]any{"chatId": chatID},
	})
	s.vk.Notify(ctx, newUserID, "У вас новый чат", s.chatURL(chatID))

	return nil
}

// RemoveParticipant удаляет участника из группового чата. Себя может убрать
// любой участник — это и есть «выйти из чата». Убрать кого-то ДРУГОГО может
// только создатель чата или участник с ролью admin — та же авторизация, что
// и у DeleteChat; раньше её тут не было вовсе (любой участник мог выкинуть
// кого угодно, включая админа) — это и есть баг, который фиксит этот метод.
// В личном чате (ровно два участника) удалять по одному смысла нет — там
// для этого есть DeleteChat.
func (s *service) RemoveParticipant(ctx context.Context, chatID, callerUserID, targetUserID string) error {
	participant, err := s.ensureParticipant(ctx, chatID, callerUserID)
	if err != nil {
		return err
	}

	if callerUserID != targetUserID {
		c, err := s.repo.GetChatByID(ctx, chatID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrChatNotFound
			}
			return err
		}
		if c.Type != repo.ChatsTypeGroup {
			return ErrNotAllowed
		}
		if c.CreatedByUserID != callerUserID && participant.Role != repo.ChatParticipantsRoleAdmin {
			return ErrNotAllowed
		}
	}

	if err := s.repo.RemoveChatParticipant(ctx, repo.RemoveChatParticipantParams{
		ChatID: chatID,
		UserID: targetUserID,
	}); err != nil {
		return err
	}

	if callerUserID != targetUserID {
		// Удалённому — чат у него просто пропадает, как при DeleteChat.
		s.hub.SendToUser(targetUserID, Event{
			Type: EventChatDeleted,
			Data: map[string]any{"chatId": chatID},
		})
	}

	// Оставшимся участникам — обновить состав в уже открытом треде.
	// broadcastToParticipants сам читает участников УЖЕ ПОСЛЕ удаления, так
	// что targetUserID среди адресатов не будет.
	s.broadcastToParticipants(ctx, chatID, Event{
		Type: EventParticipantRemoved,
		Data: map[string]any{"chatId": chatID, "userId": targetUserID},
	})

	return nil
}

// ============================================
// SSE
// ============================================

func (s *service) Subscribe(userID string) chan Event {
	return s.hub.Subscribe(userID)
}

func (s *service) Unsubscribe(userID string, ch chan Event) {
	s.hub.Unsubscribe(userID, ch)
}

// ============================================
// Утилиты
// ============================================

// ensureParticipant — центральная проверка авторизации: участвует ли
// callerUserID в чате chatID. Без неё любой залогиненный пользователь мог
// бы читать/писать в чужие чаты, зная только id.
func (s *service) ensureParticipant(ctx context.Context, chatID, callerUserID string) (repo.ChatParticipant, error) {
	p, err := s.repo.GetChatParticipant(ctx, repo.GetChatParticipantParams{
		ChatID: chatID,
		UserID: callerUserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repo.ChatParticipant{}, ErrNotParticipant
		}
		return repo.ChatParticipant{}, err
	}
	return p, nil
}

// broadcastToParticipants рассылает событие всем участникам чата, кроме
// перечисленных в exclude. Список участников читается из БД каждый раз —
// для размера чатов в этом приложении (единицы-десятки человек) это
// дешевле, чем городить отдельный кэш участников в хабе.
func (s *service) broadcastToParticipants(ctx context.Context, chatID string, event Event, exclude ...string) {
	participants, err := s.repo.ListChatParticipants(ctx, chatID)
	if err != nil {
		return
	}

	excluded := make(map[string]struct{}, len(exclude))
	for _, id := range exclude {
		excluded[id] = struct{}{}
	}

	for _, p := range participants {
		if _, skip := excluded[p.UserID]; skip {
			continue
		}
		s.hub.SendToUser(p.UserID, event)
	}
}

// notifyVK — дублирует уведомление в VK участникам чата (кто привязал
// аккаунт, см. internal/vk) без mute. Только для "новое сообщение"/"новый
// чат" — не для typing/read_receipt/message_deleted, как и браузерные
// уведомления на фронте.
func (s *service) notifyVK(ctx context.Context, chatID, text string, exclude ...string) {
	participants, err := s.repo.ListChatParticipants(ctx, chatID)
	if err != nil {
		return
	}

	excluded := make(map[string]struct{}, len(exclude))
	for _, id := range exclude {
		excluded[id] = struct{}{}
	}

	ids := make([]string, 0, len(participants))
	for _, p := range participants {
		if p.Muted {
			continue
		}
		if _, skip := excluded[p.UserID]; skip {
			continue
		}
		ids = append(ids, p.UserID)
	}
	if len(ids) == 0 {
		return
	}

	s.vk.NotifyMany(ctx, ids, text, s.chatURL(chatID))
}

func (s *service) chatURL(chatID string) string {
	return fmt.Sprintf("%s/chats?open=%s", s.frontendURL, chatID)
}

// vkMessagePreview — текст для VK-уведомления о новом сообщении. Без имени
// отправителя: у chat.service нет прав дёрнуть справочник сотрудников
// (GetUsers за gRPC требует session_token, а тут только callerUserID) —
// имя резолвит только фронт, из уже загруженного списка сотрудников.
func vkMessagePreview(body string) string {
	if body == "" {
		return "Новое сообщение в чате (вложение)"
	}
	runes := []rune(body)
	if len(runes) > 200 {
		return "Новое сообщение в чате: " + string(runes[:200]) + "…"
	}
	return "Новое сообщение в чате: " + body
}

func normalizeParticipants(chatType, callerUserID string, ids []string) ([]string, error) {
	if chatType != string(repo.ChatsTypeDirect) && chatType != string(repo.ChatsTypeGroup) {
		return nil, ErrBadChatType
	}

	all := dedupe(append([]string{callerUserID}, ids...))

	if chatType == string(repo.ChatsTypeDirect) && len(all) != 2 {
		return nil, ErrDirectTwoUsers
	}
	if len(all) < 2 {
		return nil, ErrNoParticipants
	}

	return all, nil
}

// attachmentsForMessages подтягивает вложения сразу для страницы сообщений
// одним запросом (см. ListFilesByEntityIDs) — без этого пришлось бы делать
// отдельный запрос на каждое сообщение при листинге.
func (s *service) attachmentsForMessages(ctx context.Context, messageIDs []uint64) (map[uint64][]MessageAttachment, error) {
	result := make(map[uint64][]MessageAttachment, len(messageIDs))
	if len(messageIDs) == 0 {
		return result, nil
	}

	ids := make([]string, 0, len(messageIDs))
	for _, id := range messageIDs {
		ids = append(ids, strconv.FormatUint(id, 10))
	}

	rows, err := s.repo.ListFilesByEntityIDs(ctx, repo.ListFilesByEntityIDsParams{
		EntityType: entityTypeChatMessage,
		EntityIds:  ids,
	})
	if err != nil {
		return nil, err
	}

	for _, r := range rows {
		id, err := strconv.ParseUint(r.EntityID, 10, 64)
		if err != nil {
			continue
		}
		result[id] = append(result[id], MessageAttachment{
			ID:           r.ID,
			OriginalName: r.OriginalName,
			MimeType:     r.MimeType,
			FileType:     r.FileType,
			SizeBytes:    r.SizeBytes,
		})
	}

	return result, nil
}

func dedupe(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}
