package vacationtype

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	repo "timetrack/internal/adapter/mysql/sqlc"

	"github.com/google/uuid"
)

var (
	ErrNotFound        = errors.New("тип отпуска не найден")
	ErrNameTaken       = errors.New("тип отпуска с таким именем или системным именем уже существует")
	ErrTypeInUse       = errors.New("нельзя удалить тип отпуска, который используется в заявках")
	ErrColorCodeReq    = errors.New("укажите цвет типа отпуска в формате #RRGGBB")
	ErrSystemNameEmpty = errors.New("укажите системное имя типа отпуска")
	ErrNameEmpty       = errors.New("укажите название типа отпуска")
)

type Service interface {
	GetAll(ctx context.Context) ([]repo.VacationType, error)
	GetActive(ctx context.Context) ([]repo.VacationType, error)
	GetByID(ctx context.Context, id string) (repo.VacationType, error)
	Create(ctx context.Context, req CreateVacationTypeRequest) (repo.VacationType, error)
	Update(ctx context.Context, id string, req UpdateVacationTypeRequest) (repo.VacationType, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo repo.Querier
}

func NewService(repo repo.Querier) Service {
	return &service{repo: repo}
}

func (s *service) GetAll(ctx context.Context) ([]repo.VacationType, error) {
	return s.repo.GetVacationTypes(ctx)
}

func (s *service) GetActive(ctx context.Context) ([]repo.VacationType, error) {
	return s.repo.GetActiveVacationTypes(ctx)
}

func (s *service) GetByID(ctx context.Context, id string) (repo.VacationType, error) {
	t, err := s.repo.GetVacationTypeByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repo.VacationType{}, ErrNotFound
		}
		return repo.VacationType{}, err
	}
	return t, nil
}

func (s *service) Create(ctx context.Context, req CreateVacationTypeRequest) (repo.VacationType, error) {
	if err := validate(req.Name, req.SystemName, req.ColorCode); err != nil {
		return repo.VacationType{}, err
	}

	if err := s.ensureNameFree(ctx, req.SystemName, ""); err != nil {
		return repo.VacationType{}, err
	}

	id := uuid.NewString()
	if err := s.repo.CreateVacationType(ctx, repo.CreateVacationTypeParams{
		ID:             id,
		Name:           req.Name,
		SystemName:     req.SystemName,
		ColorCode:      req.ColorCode,
		AffectsBalance: req.AffectsBalance,
		IsActive:       req.IsActive,
		SortOrder:      req.SortOrder,
	}); err != nil {
		return repo.VacationType{}, fmt.Errorf("create vacation type: %w", err)
	}

	return s.GetByID(ctx, id)
}

func (s *service) Update(ctx context.Context, id string, req UpdateVacationTypeRequest) (repo.VacationType, error) {
	if _, err := s.GetByID(ctx, id); err != nil {
		return repo.VacationType{}, err
	}

	if err := validate(req.Name, req.SystemName, req.ColorCode); err != nil {
		return repo.VacationType{}, err
	}

	if err := s.ensureNameFree(ctx, req.SystemName, id); err != nil {
		return repo.VacationType{}, err
	}

	if err := s.repo.UpdateVacationType(ctx, repo.UpdateVacationTypeParams{
		Name:           req.Name,
		SystemName:     req.SystemName,
		ColorCode:      req.ColorCode,
		AffectsBalance: req.AffectsBalance,
		IsActive:       req.IsActive,
		SortOrder:      req.SortOrder,
		ID:             id,
	}); err != nil {
		return repo.VacationType{}, fmt.Errorf("update vacation type: %w", err)
	}

	return s.GetByID(ctx, id)
}

func (s *service) Delete(ctx context.Context, id string) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}

	count, err := s.repo.CountVacationsByType(ctx, sql.NullString{String: id, Valid: true})
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrTypeInUse
	}

	if err := s.repo.DeleteVacationType(ctx, id); err != nil {
		return fmt.Errorf("delete vacation type: %w", err)
	}
	return nil
}

func (s *service) ensureNameFree(ctx context.Context, systemName, excludeID string) error {
	existing, err := s.repo.GetVacationTypeBySystemName(ctx, systemName)
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

func validate(name, systemName, colorCode string) error {
	if name == "" {
		return ErrNameEmpty
	}
	if systemName == "" {
		return ErrSystemNameEmpty
	}
	if len(colorCode) != 7 || colorCode[0] != '#' {
		return ErrColorCodeReq
	}
	return nil
}
