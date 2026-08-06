package daytype

import (
	"context"
	repo "timetrack/internal/adapter/mysql/sqlc"
)

type service struct {
	repo repo.Querier
}

type Service interface {
	GetDayTypes(ctx context.Context) (*[]repo.DayType, error)
}

func NewService(repo repo.Querier) Service {
	return &service{repo: repo}
}

func (s *service) GetDayTypes(ctx context.Context) (*[]repo.DayType, error) {
	types, err := s.repo.GetDayTypes(ctx)
	if err != nil {
		return nil, err
	}
	return &types, nil
}
