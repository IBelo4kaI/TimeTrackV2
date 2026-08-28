package news

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"

	"github.com/google/uuid"
)

var (
	ErrNotFound = errors.New("новость не найдена")
	ErrTitleReq = errors.New("укажите заголовок новости")
	ErrBodyReq  = errors.New("укажите текст новости")
)

type Service interface {
	GetAll(ctx context.Context) ([]repo.NewsPost, error)
	GetByID(ctx context.Context, id string) (repo.NewsPost, error)
	Create(ctx context.Context, req CreatePostRequest) (repo.NewsPost, error)
	Update(ctx context.Context, id string, req UpdatePostRequest) (repo.NewsPost, error)
	Delete(ctx context.Context, id string) error
	CountUnread(ctx context.Context, userID string) (int64, error)
	MarkSeen(ctx context.Context, userID string) error
}

type service struct {
	repo repo.Querier
}

func NewService(repo repo.Querier) Service {
	return &service{repo: repo}
}

func (s *service) GetAll(ctx context.Context) ([]repo.NewsPost, error) {
	return s.repo.ListNewsPosts(ctx)
}

func (s *service) GetByID(ctx context.Context, id string) (repo.NewsPost, error) {
	p, err := s.repo.GetNewsPostByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repo.NewsPost{}, ErrNotFound
		}
		return repo.NewsPost{}, err
	}
	return p, nil
}

func (s *service) Create(ctx context.Context, req CreatePostRequest) (repo.NewsPost, error) {
	if err := validate(req.Title, req.Body); err != nil {
		return repo.NewsPost{}, err
	}

	id := uuid.NewString()
	if err := s.repo.CreateNewsPost(ctx, repo.CreateNewsPostParams{
		ID:    id,
		Title: req.Title,
		Body:  req.Body,
	}); err != nil {
		return repo.NewsPost{}, fmt.Errorf("create news post: %w", err)
	}

	return s.GetByID(ctx, id)
}

func (s *service) Update(ctx context.Context, id string, req UpdatePostRequest) (repo.NewsPost, error) {
	if _, err := s.GetByID(ctx, id); err != nil {
		return repo.NewsPost{}, err
	}
	if err := validate(req.Title, req.Body); err != nil {
		return repo.NewsPost{}, err
	}

	if err := s.repo.UpdateNewsPost(ctx, repo.UpdateNewsPostParams{
		ID:    id,
		Title: req.Title,
		Body:  req.Body,
	}); err != nil {
		return repo.NewsPost{}, fmt.Errorf("update news post: %w", err)
	}

	return s.GetByID(ctx, id)
}

func (s *service) Delete(ctx context.Context, id string) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	if err := s.repo.DeleteNewsPost(ctx, id); err != nil {
		return fmt.Errorf("delete news post: %w", err)
	}
	return nil
}

// CountUnread — постов новее последней отметки пользователя. Если отметки
// ещё нет (ни разу не открывал), считаем непрочитанными все посты.
func (s *service) CountUnread(ctx context.Context, userID string) (int64, error) {
	lastSeen, err := s.lastSeenAt(ctx, userID)
	if err != nil {
		return 0, err
	}
	return s.repo.CountNewsPostsSince(ctx, lastSeen)
}

func (s *service) MarkSeen(ctx context.Context, userID string) error {
	return s.repo.UpsertNewsReadMark(ctx, userID)
}

func (s *service) lastSeenAt(ctx context.Context, userID string) (time.Time, error) {
	mark, err := s.repo.GetNewsReadMark(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	return mark.LastSeenAt, nil
}

func validate(title, body string) error {
	if title == "" {
		return ErrTitleReq
	}
	if body == "" {
		return ErrBodyReq
	}
	return nil
}
