package filecategory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	repo "timetrack/internal/adapter/mysql/sqlc"

	"github.com/google/uuid"
)

var (
	ErrNotFound       = errors.New("категория не найдена")
	ErrNameTaken      = errors.New("категория с таким именем уже существует на этом уровне")
	ErrCycle          = errors.New("нельзя переместить категорию внутрь себя или своего потомка")
	ErrParentMissing  = errors.New("родительская категория не найдена")
	ErrSystemCategory = errors.New("это системная категория, заданная кодом, — её нельзя удалить")
)

type Service interface {
	GetTree(ctx context.Context) ([]*FileCategoryNode, error)
	GetByID(ctx context.Context, id string) (repo.FileCategory, error)
	Create(ctx context.Context, req CreateFileCategoryRequest) (repo.FileCategory, error)
	Update(ctx context.Context, id string, req UpdateFileCategoryRequest) (repo.FileCategory, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo repo.Querier
}

func NewService(repo repo.Querier) Service {
	return &service{repo: repo}
}

func (s *service) GetTree(ctx context.Context) ([]*FileCategoryNode, error) {
	rows, err := s.repo.GetFileCategories(ctx)
	if err != nil {
		return nil, err
	}

	nodes := make(map[string]*FileCategoryNode, len(rows))
	var roots []*FileCategoryNode

	for _, r := range rows {
		nodes[r.ID] = toNode(r)
	}

	for _, r := range rows {
		node := nodes[r.ID]
		if r.ParentID.Valid {
			if parent, ok := nodes[r.ParentID.String]; ok {
				parent.Children = append(parent.Children, node)
				continue
			}
		}
		roots = append(roots, node)
	}

	return roots, nil
}

func (s *service) GetByID(ctx context.Context, id string) (repo.FileCategory, error) {
	c, err := s.repo.GetFileCategoryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repo.FileCategory{}, ErrNotFound
		}
		return repo.FileCategory{}, err
	}
	return c, nil
}

func (s *service) Create(ctx context.Context, req CreateFileCategoryRequest) (repo.FileCategory, error) {
	parentID := nullString(req.ParentID)

	if parentID.Valid {
		if _, err := s.repo.GetFileCategoryByID(ctx, parentID.String); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return repo.FileCategory{}, ErrParentMissing
			}
			return repo.FileCategory{}, err
		}
	}

	if err := s.ensureNameFree(ctx, parentID, req.Name, ""); err != nil {
		return repo.FileCategory{}, err
	}

	id := uuid.NewString()
	if err := s.repo.CreateFileCategory(ctx, repo.CreateFileCategoryParams{
		ID:        id,
		Name:      req.Name,
		ParentID:  parentID,
		ColorCode: nullString(req.ColorCode),
		SortOrder: req.SortOrder,
	}); err != nil {
		return repo.FileCategory{}, fmt.Errorf("create file category: %w", err)
	}

	return s.GetByID(ctx, id)
}

func (s *service) Update(ctx context.Context, id string, req UpdateFileCategoryRequest) (repo.FileCategory, error) {
	if _, err := s.GetByID(ctx, id); err != nil {
		return repo.FileCategory{}, err
	}

	parentID := nullString(req.ParentID)

	if parentID.Valid {
		if parentID.String == id {
			return repo.FileCategory{}, ErrCycle
		}
		if _, err := s.repo.GetFileCategoryByID(ctx, parentID.String); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return repo.FileCategory{}, ErrParentMissing
			}
			return repo.FileCategory{}, err
		}
		isDescendant, err := s.isDescendant(ctx, id, parentID.String)
		if err != nil {
			return repo.FileCategory{}, err
		}
		if isDescendant {
			return repo.FileCategory{}, ErrCycle
		}
	}

	if err := s.ensureNameFree(ctx, parentID, req.Name, id); err != nil {
		return repo.FileCategory{}, err
	}

	if err := s.repo.UpdateFileCategory(ctx, repo.UpdateFileCategoryParams{
		Name:      req.Name,
		ParentID:  parentID,
		ColorCode: nullString(req.ColorCode),
		SortOrder: req.SortOrder,
		ID:        id,
	}); err != nil {
		return repo.FileCategory{}, fmt.Errorf("update file category: %w", err)
	}

	return s.GetByID(ctx, id)
}

func (s *service) Delete(ctx context.Context, id string) error {
	c, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if c.SystemName.Valid {
		return ErrSystemCategory
	}
	// Дочерние категории удаляются каскадом (ON DELETE CASCADE),
	// файлы в удалённых категориях становятся без категории (ON DELETE SET NULL).
	if err := s.repo.DeleteFileCategory(ctx, id); err != nil {
		return fmt.Errorf("delete file category: %w", err)
	}
	return nil
}

// isDescendant проверяет, является ли candidateID потомком rootID.
func (s *service) isDescendant(ctx context.Context, rootID, candidateID string) (bool, error) {
	currentID := candidateID
	for i := 0; i < 1000; i++ { // защита от зацикливания на повреждённых данных
		node, err := s.repo.GetFileCategoryByID(ctx, currentID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return false, nil
			}
			return false, err
		}
		if !node.ParentID.Valid {
			return false, nil
		}
		if node.ParentID.String == rootID {
			return true, nil
		}
		currentID = node.ParentID.String
	}
	return false, nil
}

func (s *service) ensureNameFree(ctx context.Context, parentID sql.NullString, name, excludeID string) error {
	existing, err := s.repo.GetFileCategoryByParentAndName(ctx, repo.GetFileCategoryByParentAndNameParams{
		Name:     name,
		ParentID: parentID,
	})
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

func toNode(r repo.FileCategory) *FileCategoryNode {
	node := &FileCategoryNode{
		ID:        r.ID,
		Name:      r.Name,
		SortOrder: r.SortOrder,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
		Children:  []*FileCategoryNode{},
	}
	if r.ParentID.Valid {
		v := r.ParentID.String
		node.ParentID = &v
	}
	if r.ColorCode.Valid {
		v := r.ColorCode.String
		node.ColorCode = &v
	}
	if r.SystemName.Valid {
		v := r.SystemName.String
		node.SystemName = &v
	}
	return node
}

func nullString(v *string) sql.NullString {
	if v == nil || *v == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *v, Valid: true}
}
