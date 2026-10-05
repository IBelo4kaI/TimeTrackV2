package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	repo "timetrack/internal/adapter/mysql/sqlc"
)

// EntityTypeChatMessage — вложения сообщений чата (см. internal/chat): видят
// только участники чата, права ".all" это не расширяют (личная переписка).
const EntityTypeChatMessage = "chat_message"

// RestrictedEntityPermissions — типы сущностей с ограниченным доступом к
// файлам: тип -> сущность права ".all" (чужое открывает <entity>.all:<action>).
// Остальные типы (вложения чата, файлы без привязки и т.п.) открыты любому с
// базовым files:*, как и раньше — чтобы закрыть новый тип, достаточно добавить
// его сюда и в entityOwner.
var RestrictedEntityPermissions = map[string]string{
	"vacation":   "vacation",
	"sick_leave": "sick_leaves",
	"receipt":    "receipts",
}

// HasAllFunc — есть ли у вызывающего право на чужие записи этой сущности.
type HasAllFunc func(entity string) bool

func (s *FileService) entityOwner(ctx context.Context, entityType, entityID string) (string, error) {
	switch entityType {
	case "vacation":
		return s.repo.GetVacationOwnerID(ctx, entityID)
	case "receipt":
		return s.repo.GetReceiptOwnerID(ctx, entityID)
	case "sick_leave":
		return s.repo.GetSickLeaveOwnerID(ctx, entityID)
	}
	return "", fmt.Errorf("unsupported entity type: %s", entityType)
}

// CanAccessEntity — доступ к файлам сущности: открытые типы — всем, ограниченные
// — владельцу сущности или с <entity>.all, вложения чата — участникам чата.
func (s *FileService) CanAccessEntity(ctx context.Context, entityType, entityID, userID string, hasAll HasAllFunc) (bool, error) {
	if entityType == EntityTypeChatMessage {
		messageID, err := strconv.ParseUint(entityID, 10, 64)
		if err != nil {
			return false, nil
		}
		return s.repo.IsChatMessageParticipant(ctx, repo.IsChatMessageParticipantParams{ID: messageID, UserID: userID})
	}

	allEntity, restricted := RestrictedEntityPermissions[entityType]
	if !restricted {
		return true, nil
	}

	owner, err := s.entityOwner(ctx, entityType, entityID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	if err == nil && owner == userID {
		return true, nil
	}
	return hasAll(allEntity), nil
}

// CanAccessFile — доступ к файлу по его привязкам: достаточно, чтобы
// разрешила хотя бы одна. Файл без привязок открыт (как и раньше).
func (s *FileService) CanAccessFile(ctx context.Context, fileID, userID string, hasAll HasAllFunc) (bool, error) {
	refs, err := s.repo.ListEntityRefsByFile(ctx, fileID)
	if err != nil {
		return false, fmt.Errorf("list entity refs: %w", err)
	}
	if len(refs) == 0 {
		return true, nil
	}

	for _, r := range refs {
		ok, err := s.CanAccessEntity(ctx, r.EntityType, r.EntityID, userID, hasAll)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}
