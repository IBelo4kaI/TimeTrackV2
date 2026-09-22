package receiptcategory

import (
	"context"
	"errors"
	"strings"
	repo "timetrack/internal/adapter/mysql/sqlc"
)

var (
	ErrNameRequired      = errors.New("не указано название категории")
	ErrKeywordRequired   = errors.New("не указано ключевое слово")
	ErrCategoryDuplicate = errors.New("категория с таким названием уже существует")
	ErrKeywordDuplicate  = errors.New("такое ключевое слово уже есть в словаре")
)

type Service interface {
	ListCategories(ctx context.Context) ([]repo.Category, error)
	CreateCategory(ctx context.Context, name string) (repo.Category, error)

	ListKeywords(ctx context.Context) ([]repo.KeywordCategory, error)
	CreateKeyword(ctx context.Context, keyword string, categoryID int32) (repo.KeywordCategory, error)

	Classify(ctx context.Context, sellerInn string, items []ItemForClassify) (ClassifyResult, error)
	// Preview — та же классификация, но без самообучения (см. classify.go)
	// — для показа определившейся категории ДО сохранения чека.
	Preview(ctx context.Context, sellerInn string, items []ItemForClassify) (ClassifyResult, error)
	// SetOverride — сотрудник вручную поправил категорию чека: запоминаем
	// маппинг ИНН -> категория с source='user_override' (перезаписывает
	// прежний, каким бы он ни был), чтобы будущие чеки от этого продавца
	// сразу попадали в шаг 1 алгоритма с правильной категорией.
	SetOverride(ctx context.Context, sellerInn string, categoryID int32) error
}

type service struct {
	repo *repo.Queries
}

func NewService(repo *repo.Queries) Service {
	return &service{repo: repo}
}

func (s *service) ListCategories(ctx context.Context) ([]repo.Category, error) {
	return s.repo.ListCategories(ctx)
}

func (s *service) CreateCategory(ctx context.Context, name string) (repo.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return repo.Category{}, ErrNameRequired
	}

	existing, err := s.repo.ListCategories(ctx)
	if err != nil {
		return repo.Category{}, err
	}
	for _, c := range existing {
		if strings.EqualFold(c.Name, name) {
			return repo.Category{}, ErrCategoryDuplicate
		}
	}

	id, err := s.repo.CreateCategory(ctx, name)
	if err != nil {
		return repo.Category{}, err
	}

	return repo.Category{ID: int32(id), Name: name, IsSystem: false}, nil
}

func (s *service) ListKeywords(ctx context.Context) ([]repo.KeywordCategory, error) {
	return s.repo.ListKeywordCategories(ctx)
}

func (s *service) CreateKeyword(ctx context.Context, keyword string, categoryID int32) (repo.KeywordCategory, error) {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return repo.KeywordCategory{}, ErrKeywordRequired
	}

	existing, err := s.repo.ListKeywordCategories(ctx)
	if err != nil {
		return repo.KeywordCategory{}, err
	}
	for _, k := range existing {
		if k.Keyword == keyword {
			return repo.KeywordCategory{}, ErrKeywordDuplicate
		}
	}

	if err := s.repo.CreateKeyword(ctx, repo.CreateKeywordParams{
		Keyword:    keyword,
		CategoryID: categoryID,
	}); err != nil {
		return repo.KeywordCategory{}, err
	}

	return repo.KeywordCategory{Keyword: keyword, CategoryID: categoryID, IsSystem: false}, nil
}

func (s *service) SetOverride(ctx context.Context, sellerInn string, categoryID int32) error {
	if sellerInn == "" {
		return nil
	}
	return s.repo.UpsertMerchantCategory(ctx, repo.UpsertMerchantCategoryParams{
		Inn:        sellerInn,
		CategoryID: categoryID,
		Source:     "user_override",
	})
}
