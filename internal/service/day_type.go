package service

import (
	"context"
	repo "timetrack/internal/adapter/mysql/sqlc"
)

type dayTypeService struct {
	repo repo.Querier
}

type DayTypeService interface {
	GetDayTypes(ctx context.Context) (*[]repo.DayType, error)
}

func NewDayTypeService(repo repo.Querier) DayTypeService {
	return &dayTypeService{repo: repo}
}

func (s *dayTypeService) GetDayTypes(ctx context.Context) (*[]repo.DayType, error) {
	types, err := s.repo.GetDayTypes(ctx)
	if err != nil {
		return nil, err
	}
	return &types, nil
}
