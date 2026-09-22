package receiptcategory

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
)

var (
	ErrNameRequired      = errors.New("не указано название категории")
	ErrKeywordRequired   = errors.New("не указано ключевое слово")
	ErrCategoryDuplicate = errors.New("категория с таким названием уже существует")
	ErrKeywordDuplicate  = errors.New("такое ключевое слово уже есть в словаре")
	ErrSellerInnRequired = errors.New("не указан ИНН продавца")
	ErrCategoryNotFound  = errors.New("категория не найдена")
)

type Service interface {
	ListCategories(ctx context.Context) ([]repo.Category, error)
	CreateCategory(ctx context.Context, name string) (repo.Category, error)
	RenameCategory(ctx context.Context, id int32, name string) (repo.Category, error)

	ListKeywords(ctx context.Context) ([]repo.KeywordCategory, error)
	CreateKeyword(ctx context.Context, keyword string, categoryID int32) (repo.KeywordCategory, error)

	Classify(ctx context.Context, sellerInn string, items []ItemForClassify) (ClassifyResult, error)
	// Preview — та же классификация, но без самообучения (см. classify.go)
	// — для показа определившейся категории ДО сохранения чека.
	Preview(ctx context.Context, sellerInn string, items []ItemForClassify) (ClassifyResult, error)

	// ListMerchants — словарь "ИНН продавца -> категория" целиком, для
	// экрана настроек "Категории и слова" (вкладка "Продавцы").
	ListMerchants(ctx context.Context) ([]repo.ListMerchantCategoriesRow, error)
	// UpdateMerchant — сотрудник вручную поправил категорию продавца (через
	// экран "Продавцы" ИЛИ через категорию одного чека — оба пути идут
	// сюда, см. receipt.SetCategory): запоминает маппинг ИНН -> категория с
	// source='user_override' (перезаписывает прежний, каким бы он ни был) и
	// сразу переносит новую категорию на ВСЕ уже сохранённые чеки этого
	// продавца, а не только на будущие — возвращает, сколько чеков задело.
	UpdateMerchant(ctx context.Context, sellerInn string, categoryID int32) (updatedReceipts int64, err error)
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

func (s *service) RenameCategory(ctx context.Context, id int32, name string) (repo.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return repo.Category{}, ErrNameRequired
	}

	existing, err := s.repo.ListCategories(ctx)
	if err != nil {
		return repo.Category{}, err
	}

	var found *repo.Category
	for i, c := range existing {
		if c.ID == id {
			found = &existing[i]
			continue
		}
		if strings.EqualFold(c.Name, name) {
			return repo.Category{}, ErrCategoryDuplicate
		}
	}
	if found == nil {
		return repo.Category{}, ErrCategoryNotFound
	}

	if err := s.repo.RenameCategory(ctx, repo.RenameCategoryParams{
		Name: name,
		ID:   id,
	}); err != nil {
		return repo.Category{}, err
	}

	found.Name = name
	return *found, nil
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

func (s *service) ListMerchants(ctx context.Context) ([]repo.ListMerchantCategoriesRow, error) {
	return s.repo.ListMerchantCategories(ctx)
}

func (s *service) UpdateMerchant(ctx context.Context, sellerInn string, categoryID int32) (int64, error) {
	if sellerInn == "" {
		return 0, ErrSellerInnRequired
	}

	if err := s.repo.UpsertMerchantCategory(ctx, repo.UpsertMerchantCategoryParams{
		Inn:        sellerInn,
		CategoryID: categoryID,
		Source:     "user_override",
	}); err != nil {
		return 0, err
	}

	// Связь продавец -> категория поменялась — переносим новую категорию и
	// на все уже сохранённые чеки этого продавца, а не только на будущие.
	return s.repo.UpdateReceiptsCategoryBySellerInn(ctx, repo.UpdateReceiptsCategoryBySellerInnParams{
		CategoryID: sql.NullInt32{Int32: categoryID, Valid: true},
		UpdatedAt:  time.Now().UTC(),
		SellerInn:  sellerInn,
	})
}
