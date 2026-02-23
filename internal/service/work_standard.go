package service

import (
	"context"
	"database/sql"
	"strconv"

	repo "timetrack/internal/adapter/mysql/sqlc"
)

type WorkStandardService struct {
	repo repo.Querier
}

func NewWorkStandardService(repo repo.Querier) *WorkStandardService {
	return &WorkStandardService{repo: repo}
}

// CreateWorkStandard создает новый стандарт работы
func (s *WorkStandardService) CreateWorkStandard(ctx context.Context, params repo.CreateWorkStandardParams) error {
	return s.repo.CreateWorkStandard(ctx, params)
}

// GetWorkStandardById получает стандарт работы по ID
func (s *WorkStandardService) GetWorkStandardById(ctx context.Context, id string) (repo.WorkStandard, error) {
	return s.repo.GetWorkStandardsById(ctx, id)
}

// GetWorkStandardsByMonth получает стандарты работы по месяцу и году
func (s *WorkStandardService) GetWorkStandardsByMonth(ctx context.Context, month, year int32) ([]repo.WorkStandard, error) {
	params := repo.GetWorkStandardsByMonthParams{
		Month: month,
		Year:  year,
	}
	return s.repo.GetWorkStandardsByMonth(ctx, params)
}

// GetWorkStandardByMonthAndGender получает стандарт работы по месяцу, году, полу и user_id
func (s *WorkStandardService) GetWorkStandardByMonthAndGender(ctx context.Context, month, year, gender int32, userID string) (repo.WorkStandard, error) {
	params := repo.GetWorkStandardsByMonthAndGenderIdParams{
		Month:  month,
		Year:   year,
		Gender: gender,
		UserID: sql.NullString{
			String: userID,
			Valid:  userID != "",
		},
	}
	return s.repo.GetWorkStandardsByMonthAndGenderId(ctx, params)
}

// GetWorkStandardsByYear получает стандарты работы по году
func (s *WorkStandardService) GetWorkStandardsByYear(ctx context.Context, year int32) ([]repo.WorkStandard, error) {
	workStandards, err := s.repo.GetWorkStandardsByYear(ctx, year)
	if err != nil {
		return nil, err
	}

	return workStandards, nil
}

// GetWorkStandardsByYearGrouped получает стандарты работы по году, сгруппированные по месяцам и полу
func (s *WorkStandardService) GetWorkStandardsByYearGrouped(ctx context.Context, year int32) (map[string]map[string]repo.WorkStandard, error) {
	workStandards, err := s.repo.GetWorkStandardsByYear(ctx, year)
	if err != nil {
		return nil, err
	}

	// Создаем структуру: месяц -> пол -> стандарт
	grouped := make(map[string]map[string]repo.WorkStandard)

	// Названия месяцев
	monthNames := map[int32]string{
		1: "january", 2: "february", 3: "march", 4: "april",
		5: "may", 6: "june", 7: "july", 8: "august",
		9: "september", 10: "october", 11: "november", 12: "december",
	}

	// Названия полов (предполагаем: 1 = men, 2 = women)
	genderNames := map[int32]string{
		1: "men",
		2: "women",
	}

	for _, standard := range workStandards {
		monthName, ok := monthNames[standard.Month]
		if !ok {
			continue // Пропускаем невалидные месяцы
		}

		genderName, ok := genderNames[standard.Gender]
		if !ok {
			// Если пол не 1 или 2, используем числовое значение как строку
			genderName = strconv.Itoa(int(standard.Gender))
		}

		// Инициализируем map для месяца, если нужно
		if grouped[monthName] == nil {
			grouped[monthName] = make(map[string]repo.WorkStandard)
		}

		// Сохраняем стандарт
		grouped[monthName][genderName] = standard
	}

	return grouped, nil
}

// UpdateWorkStandard обновляет стандарт работы
func (s *WorkStandardService) UpdateWorkStandard(ctx context.Context, params repo.UpdateWorkStandardParams) error {
	return s.repo.UpdateWorkStandard(ctx, params)
}

// DeleteWorkStandard удаляет стандарт работы
func (s *WorkStandardService) DeleteWorkStandard(ctx context.Context, id string) error {
	return s.repo.DeleteWorkStandard(ctx, id)
}
