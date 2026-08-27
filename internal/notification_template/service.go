// Package notificationtemplate — готовые шаблоны для ручной рассылки
// уведомлений админом сотрудникам (заголовок+текст, сохранённые под именем,
// чтобы не набирать заново каждый раз) — см. internal/notification.Service.SendManual,
// который их непосредственно не знает: фронт сам подставляет title/message
// шаблона в форму отправки.
package notificationtemplate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	repo "timetrack/internal/adapter/mysql/sqlc"

	"github.com/google/uuid"
)

var (
	ErrNotFound  = errors.New("шаблон не найден")
	ErrNameTaken = errors.New("шаблон с таким именем уже существует")
	ErrNameEmpty = errors.New("укажите имя шаблона")
	ErrTitleReq  = errors.New("укажите заголовок уведомления")
)

type Service interface {
	GetAll(ctx context.Context) ([]repo.NotificationTemplate, error)
	GetByID(ctx context.Context, id string) (repo.NotificationTemplate, error)
	Create(ctx context.Context, req CreateNotificationTemplateRequest) (repo.NotificationTemplate, error)
	Update(ctx context.Context, id string, req UpdateNotificationTemplateRequest) (repo.NotificationTemplate, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo repo.Querier
}

func NewService(repo repo.Querier) Service {
	return &service{repo: repo}
}

func (s *service) GetAll(ctx context.Context) ([]repo.NotificationTemplate, error) {
	return s.repo.ListNotificationTemplates(ctx)
}

func (s *service) GetByID(ctx context.Context, id string) (repo.NotificationTemplate, error) {
	t, err := s.repo.GetNotificationTemplateByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repo.NotificationTemplate{}, ErrNotFound
		}
		return repo.NotificationTemplate{}, err
	}
	return t, nil
}

func (s *service) Create(ctx context.Context, req CreateNotificationTemplateRequest) (repo.NotificationTemplate, error) {
	if err := validate(req.Name, req.Title); err != nil {
		return repo.NotificationTemplate{}, err
	}
	if err := s.ensureNameFree(ctx, req.Name, ""); err != nil {
		return repo.NotificationTemplate{}, err
	}

	id := uuid.NewString()
	if err := s.repo.CreateNotificationTemplate(ctx, repo.CreateNotificationTemplateParams{
		ID:      id,
		Name:    req.Name,
		Title:   req.Title,
		Message: req.Message,
	}); err != nil {
		return repo.NotificationTemplate{}, fmt.Errorf("create notification template: %w", err)
	}

	return s.GetByID(ctx, id)
}

func (s *service) Update(ctx context.Context, id string, req UpdateNotificationTemplateRequest) (repo.NotificationTemplate, error) {
	if _, err := s.GetByID(ctx, id); err != nil {
		return repo.NotificationTemplate{}, err
	}
	if err := validate(req.Name, req.Title); err != nil {
		return repo.NotificationTemplate{}, err
	}
	if err := s.ensureNameFree(ctx, req.Name, id); err != nil {
		return repo.NotificationTemplate{}, err
	}

	if err := s.repo.UpdateNotificationTemplate(ctx, repo.UpdateNotificationTemplateParams{
		ID:      id,
		Name:    req.Name,
		Title:   req.Title,
		Message: req.Message,
	}); err != nil {
		return repo.NotificationTemplate{}, fmt.Errorf("update notification template: %w", err)
	}

	return s.GetByID(ctx, id)
}

func (s *service) Delete(ctx context.Context, id string) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	if err := s.repo.DeleteNotificationTemplate(ctx, id); err != nil {
		return fmt.Errorf("delete notification template: %w", err)
	}
	return nil
}

func (s *service) ensureNameFree(ctx context.Context, name, excludeID string) error {
	existing, err := s.repo.GetNotificationTemplateByName(ctx, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	if existing.ID != excludeID {
		return ErrNameTaken
	}
	return nil
}

func validate(name, title string) error {
	if name == "" {
		return ErrNameEmpty
	}
	if title == "" {
		return ErrTitleReq
	}
	return nil
}
