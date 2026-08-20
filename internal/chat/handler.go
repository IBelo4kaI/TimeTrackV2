package chat

import (
	"errors"
	"net/http"
	"strconv"
	"timetrack/internal/response"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/sse"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return Handler{service: service}
}

// callerID достаёт user_id, положенный middleware.Require в Locals после
// проверки сессии — так же, как это делают остальные хендлеры в проекте.
func callerID(c fiber.Ctx) string {
	id, _ := c.Locals("user_id").(string)
	return id
}

// ============================================
// SSE-стрим
// ============================================

// Stream — один SSE-коннекшен на пользователя (не на чат): мультиплексирует
// события по всем чатам, в которых он участвует. Держит соединение открытым
// до дисконнекта клиента.
func (h Handler) Stream(c fiber.Ctx, stream *sse.Stream) error {
	userID := callerID(c)
	if userID == "" {
		return fiber.ErrUnauthorized
	}

	events := h.service.Subscribe(userID)
	defer h.service.Unsubscribe(userID, events)

	for {
		select {
		case event, ok := <-events:
			if !ok {
				return nil
			}
			if err := stream.Event(sse.Event{
				Name: string(event.Type),
				Data: event.Data,
			}); err != nil {
				return err
			}
		case <-stream.Done():
			return stream.Err()
		}
	}
}

// ============================================
// Чаты
// ============================================

func (h Handler) ListMyChats(c fiber.Ctx) error {
	chats, err := h.service.ListMyChats(c.RequestCtx(), callerID(c))
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, chats)
}

func (h Handler) CreateChat(c fiber.Ctx) error {
	var body CreateChatRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	chat, err := h.service.CreateChat(c.RequestCtx(), callerID(c), body)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, chat)
}

type getOrCreateEntityChatRequest struct {
	OtherUserIDs []string `json:"otherUserIds"`
}

func (h Handler) GetOrCreateEntityChat(c fiber.Ctx) error {
	entityType := c.Params("entityType")
	entityID := c.Params("entityId")
	if entityType == "" || entityID == "" {
		return response.BadRequest(c)
	}

	var body getOrCreateEntityChatRequest
	// Тело необязательно (find-or-create без новых участников тоже валиден).
	_ = c.Bind().Body(&body)

	chat, err := h.service.GetOrCreateEntityChat(c.RequestCtx(), entityType, entityID, callerID(c), body.OtherUserIDs)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, chat)
}

func (h Handler) GetChat(c fiber.Ctx) error {
	chat, err := h.service.GetChat(c.RequestCtx(), c.Params("id"), callerID(c))
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, chat)
}

type renameChatRequest struct {
	Name string `json:"name"`
}

func (h Handler) RenameChat(c fiber.Ctx) error {
	var body renameChatRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	if err := h.service.RenameChat(c.RequestCtx(), c.Params("id"), callerID(c), body.Name); err != nil {
		return mapError(c, err)
	}
	return response.Updated(c)
}

func (h Handler) DeleteChat(c fiber.Ctx) error {
	if err := h.service.DeleteChat(c.RequestCtx(), c.Params("id"), callerID(c)); err != nil {
		return mapError(c, err)
	}
	return response.Deleted(c)
}

// ============================================
// Сообщения
// ============================================

func (h Handler) ListMessages(c fiber.Ctx) error {
	var beforeID *uint64
	if raw := c.Query("beforeId"); raw != "" {
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return response.BadRequest(c)
		}
		beforeID = &v
	}

	limit := int32(0)
	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return response.BadRequest(c)
		}
		limit = int32(v)
	}

	messages, err := h.service.ListMessages(c.RequestCtx(), c.Params("id"), callerID(c), beforeID, limit)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, messages)
}

func (h Handler) SendMessage(c fiber.Ctx) error {
	var body SendMessageRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	message, err := h.service.SendMessage(c.RequestCtx(), c.Params("id"), callerID(c), body.Body)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, message)
}

// SendFileMessage godoc
// POST /chats/:id/messages/file
// multipart form: file (required), body (optional caption)
func (h Handler) SendFileMessage(c fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "файл не найден в запросе"))
	}

	message, err := h.service.SendFileMessage(c.RequestCtx(), c.Params("id"), callerID(c), c.FormValue("body"), fileHeader)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, message)
}

func (h Handler) DeleteMessage(c fiber.Ctx) error {
	messageID, err := strconv.ParseUint(c.Params("messageId"), 10, 64)
	if err != nil {
		return response.BadRequest(c)
	}

	if err := h.service.DeleteMessage(c.RequestCtx(), messageID, callerID(c)); err != nil {
		return mapError(c, err)
	}
	return response.Deleted(c)
}

func (h Handler) MarkRead(c fiber.Ctx) error {
	var body MarkReadRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	if err := h.service.MarkRead(c.RequestCtx(), c.Params("id"), callerID(c), body.MessageID); err != nil {
		return mapError(c, err)
	}
	return response.Updated(c)
}

func (h Handler) Typing(c fiber.Ctx) error {
	if err := h.service.Typing(c.RequestCtx(), c.Params("id"), callerID(c)); err != nil {
		return mapError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}

// ============================================
// Участники
// ============================================

func (h Handler) ListParticipants(c fiber.Ctx) error {
	participants, err := h.service.ListParticipants(c.RequestCtx(), c.Params("id"), callerID(c))
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, participants)
}

func (h Handler) AddParticipant(c fiber.Ctx) error {
	var body AddParticipantRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}
	if body.UserID == "" {
		return response.BadRequest(c)
	}

	if err := h.service.AddParticipant(c.RequestCtx(), c.Params("id"), callerID(c), body.UserID, body.Role); err != nil {
		return mapError(c, err)
	}
	return response.Created(c)
}

func (h Handler) RemoveParticipant(c fiber.Ctx) error {
	if err := h.service.RemoveParticipant(c.RequestCtx(), c.Params("id"), callerID(c), c.Params("userId")); err != nil {
		return mapError(c, err)
	}
	return response.Deleted(c)
}

// ============================================

func mapError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrChatNotFound), errors.Is(err, ErrMessageNotFound):
		return response.Error(c, http.StatusNotFound, err)
	case errors.Is(err, ErrNotParticipant), errors.Is(err, ErrNotOwnMessage), errors.Is(err, ErrNotAllowed):
		return response.Error(c, http.StatusForbidden, err)
	case errors.Is(err, ErrBadChatType), errors.Is(err, ErrNoParticipants),
		errors.Is(err, ErrEmptyBody), errors.Is(err, ErrDirectTwoUsers):
		return response.Error(c, http.StatusBadRequest, err)
	default:
		return response.Error(c, http.StatusInternalServerError, err)
	}
}
