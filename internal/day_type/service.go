package daytype

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	repo "timetrack/internal/adapter/mysql/sqlc"
)

var (
	ErrNotFound        = errors.New("тип дня не найден")
	ErrNameTaken       = errors.New("тип дня с таким именем или системным именем уже существует")
	ErrTypeInUse       = errors.New("нельзя удалить тип дня, который используется в табелях или календаре")
	ErrSystemReserved  = errors.New("нельзя удалить системный тип дня, от него зависит логика приложения")
	ErrColorCodeReq    = errors.New("укажите цвет типа дня в формате #RRGGBB")
	ErrSystemNameEmpty = errors.New("укажите системное имя типа дня")
	ErrNameEmpty       = errors.New("укажите название типа дня")
)

// reservedSystemNames — system_name, захардкоженные в логике бэка и фронта
// (vacation/service.go, sick_leave/service.go, календарь) — удалять нельзя.
var reservedSystemNames = map[string]bool{
	"work":     true,
	"vacation": true,
	"medical":  true,
	"decree":   true,
	"time-off": true,
}

type Service interface {
	GetAll(ctx context.Context) ([]repo.DayType, error)
	GetByID(ctx context.Context, id string) (repo.DayType, error)
	Create(ctx context.Context, req CreateDayTypeRequest) (repo.DayType, error)
	Update(ctx context.Context, id string, req UpdateDayTypeRequest) (repo.DayType, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo repo.Querier
}

func NewService(repo repo.Querier) Service {
	return &service{repo: repo}
}

func (s *service) GetAll(ctx context.Context) ([]repo.DayType, error) {
	return s.repo.GetDayTypes(ctx)
}

func (s *service) GetByID(ctx context.Context, id string) (repo.DayType, error) {
	t, err := s.repo.GetDayTypeByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repo.DayType{}, ErrNotFound
		}
		return repo.DayType{}, err
	}
	return t, nil
}

func (s *service) Create(ctx context.Context, req CreateDayTypeRequest) (repo.DayType, error) {
	if err := validate(req.Name, req.SystemName, req.ColorCode); err != nil {
		return repo.DayType{}, err
	}

	if err := s.ensureNameFree(ctx, req.SystemName, ""); err != nil {
		return repo.DayType{}, err
	}

	if err := s.repo.CreateDayType(ctx, repo.CreateDayTypeParams{
		Name:            req.Name,
		SystemName:      req.SystemName,
		IsWorkDay:       req.IsWorkDay,
		AffectsVacation: req.AffectsVacation,
		IsUserSelect:    req.IsUserSelect,
		ColorCode:       req.ColorCode,
	}); err != nil {
		return repo.DayType{}, fmt.Errorf("create day type: %w", err)
	}

	created, err := s.repo.GetDayTypeBySystemName(ctx, req.SystemName)
	if err != nil {
		return repo.DayType{}, err
	}
	return created, nil
}

func (s *service) Update(ctx context.Context, id string, req UpdateDayTypeRequest) (repo.DayType, error) {
	existing, err := s.GetByID(ctx, id)
	if err != nil {
		return repo.DayType{}, err
	}

	if err := validate(req.Name, existing.SystemName, req.ColorCode); err != nil {
		return repo.DayType{}, err
	}

	if err := s.repo.UpdateDayType(ctx, repo.UpdateDayTypeParams{
		Name:            req.Name,
		SystemName:      existing.SystemName,
		IsWorkDay:       req.IsWorkDay,
		AffectsVacation: req.AffectsVacation,
		IsUserSelect:    req.IsUserSelect,
		ColorCode:       req.ColorCode,
		ID:              id,
	}); err != nil {
		return repo.DayType{}, fmt.Errorf("update day type: %w", err)
	}

	return s.GetByID(ctx, id)
}

func (s *service) Delete(ctx context.Context, id string) error {
	existing, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if reservedSystemNames[existing.SystemName] {
		return ErrSystemReserved
	}

	entriesCount, err := s.repo.CountUserTimeEntriesByDayType(ctx, id)
	if err != nil {
		return err
	}
	eventsCount, err := s.repo.CountCalendarEventsByDayType(ctx, id)
	if err != nil {
		return err
	}
	if entriesCount > 0 || eventsCount > 0 {
		return ErrTypeInUse
	}

	if err := s.repo.DeleteDayType(ctx, id); err != nil {
		return fmt.Errorf("delete day type: %w", err)
	}
	return nil
}

func (s *service) ensureNameFree(ctx context.Context, systemName, excludeID string) error {
	existing, err := s.repo.GetDayTypeBySystemName(ctx, systemName)
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
