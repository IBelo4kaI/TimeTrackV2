package usertimeentry

import (
	"context"
	"database/sql"
	"strconv"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/date"
	"timetrack/internal/parser"

	"golang.org/x/sync/errgroup"
)

type service struct {
	repo *repo.Queries
	db   *sql.DB
}

type Service interface {
	CreateUserTimeEntry(ctx context.Context, entries []repo.CreateUserTimeEntryParams) error
	DeleteUserTimeEntries(ctx context.Context, prm repo.DeleteUserTimeEntriesParams) error
	UpdateUserTimeEntries(ctx context.Context, entries []repo.UpdateUserTimeEntryParams) error
	GetStatisticsHoursByMonth(ctx context.Context, userId string, month int, year int, gender int) (*HoursStatisticResponse, error)
	GetStatisticsWorkDaysByMonth(ctx context.Context, userId string, month int, year int, gender int) (*WorkDaysStatisticResponse, error)
	GetCountDaysByMonthWithSystemName(ctx context.Context, userId string, month int, year int, gender int, systemName string) (*CountDaysResponse, error)
	GetVacationStatistics(ctx context.Context, userId string, year int) (*VacationStatisticsResponse, error)
	GetReportStatistics(ctx context.Context, userId string, month int, year int, gender int) (*ReportStatisticsResponse, error)
}

func NewService(repo *repo.Queries, db *sql.DB) Service {
	return &service{repo: repo, db: db}
}

func (s *service) CreateUserTimeEntry(ctx context.Context, entries []repo.CreateUserTimeEntryParams) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)

	for _, entry := range entries {
		err = qtx.CreateUserTimeEntry(ctx, entry)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *service) UpdateUserTimeEntries(ctx context.Context, entries []repo.UpdateUserTimeEntryParams) error {

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)

	for _, entry := range entries {
		err = qtx.UpdateUserTimeEntry(ctx, entry)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *service) DeleteUserTimeEntries(ctx context.Context, prm repo.DeleteUserTimeEntriesParams) error {
	return s.repo.DeleteUserTimeEntries(ctx, prm)
}

// getWorkStandardWithPriority получает норму работы с приоритетом индивидуальных данных
// Сначала пытается получить индивидуальную норму для пользователя, если не найдена - общую норму
func (s *service) getWorkStandardWithPriority(ctx context.Context, userId string, month, year, gender int32) (repo.WorkStandard, error) {
	// Сначала пытаемся получить индивидуальную норму для пользователя
	standard, err := s.repo.GetWorkStandardsByMonthAndGenderIdAndUserId(ctx, repo.GetWorkStandardsByMonthAndGenderIdAndUserIdParams{
		Month:  month,
		Year:   year,
		Gender: gender,
		UserID: sql.NullString{
			String: userId,
			Valid:  true,
		},
	})

	// Если индивидуальная норма не найдена, получаем общую норму
	if err != nil && err == sql.ErrNoRows {
		standard, err = s.repo.GetWorkStandardsByMonthAndGenderId(ctx, repo.GetWorkStandardsByMonthAndGenderIdParams{
			Month:  month,
			Year:   year,
			Gender: gender,
		})
	}

	return standard, err
}

func (s *service) GetStatisticsHoursByMonth(ctx context.Context, userId string, month int, year int, gender int) (*HoursStatisticResponse, error) {
	firstDayOfMonth := date.FirstDayOfMonth(month, year)

	totalHours, err := s.repo.GetTotalHoursByMonth(ctx, repo.GetTotalHoursByMonthParams{UserID: userId, Year: firstDayOfMonth, Month: firstDayOfMonth})

	if err != nil {
		return nil, err
	}

	standard, err := s.getWorkStandardWithPriority(ctx, userId, int32(month), int32(year), int32(gender))
	if err != nil {
		if err == sql.ErrNoRows {
			return &HoursStatisticResponse{
				TotalHours:    parser.InterfaceToFloat32(totalHours),
				StandardHours: 0,
			}, nil
		}
		return nil, err
	}

	return &HoursStatisticResponse{
		TotalHours:    parser.InterfaceToFloat32(totalHours),
		StandardHours: standard.StandardHours,
	}, nil
}

func (s *service) GetStatisticsWorkDaysByMonth(ctx context.Context, userId string, month int, year int, gender int) (*WorkDaysStatisticResponse, error) {
	firstDayOfMonth := date.FirstDayOfMonth(month, year)

	totalDays, err := s.repo.GetWorkDaysByMonth(ctx, repo.GetWorkDaysByMonthParams{UserID: userId, Month: firstDayOfMonth, Year: firstDayOfMonth})

	if err != nil {
		return nil, err
	}

	standard, err := s.getWorkStandardWithPriority(ctx, userId, int32(month), int32(year), int32(gender))
	if err != nil {
		if err == sql.ErrNoRows {
			return &WorkDaysStatisticResponse{
				TotalWorkDays:    parser.InterfaceToInt64(totalDays),
				StandardWorkDays: 0,
			}, nil
		}
		return nil, err
	}

	return &WorkDaysStatisticResponse{
		TotalWorkDays:    parser.InterfaceToInt64(totalDays),
		StandardWorkDays: standard.StandardDays,
	}, nil
}

func (s *service) GetCountDaysByMonthWithSystemName(ctx context.Context, userId string, month int, year int, gender int, systemName string) (*CountDaysResponse, error) {
	firstDayOfMonth := date.FirstDayOfMonth(month, year)

	countDays, err := s.repo.GetTotalDaysByMonthWithSystemName(ctx, repo.GetTotalDaysByMonthWithSystemNameParams{
		UserID:     userId,
		Year:       firstDayOfMonth,
		Month:      firstDayOfMonth,
		SystemName: systemName,
	})

	if err != nil {
		return nil, err
	}

	return &CountDaysResponse{
		Count: parser.InterfaceToInt64(countDays),
	}, nil
}

func (s *service) GetVacationStatistics(ctx context.Context, userId string, year int) (*VacationStatisticsResponse, error) {
	// Получаем использованные дни отпуска за год
	firstDayOfYear := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	usedVacationDaysInterface, err := s.repo.GetVacationDaysByYear(ctx, repo.GetVacationDaysByYearParams{
		UserID: userId,
		Year:   firstDayOfYear,
	})

	if err != nil {
		return nil, err
	}

	usedVacationDays := parser.InterfaceToInt64(usedVacationDaysInterface)

	// Получаем общее количество отпускных дней из настроек системы
	setting, err := s.repo.GetSystemSettingByKey(ctx, "vacation_duration")
	if err != nil {
		// Если настройка не найдена, используем значение по умолчанию
		if err == sql.ErrNoRows {
			return &VacationStatisticsResponse{
				UsedVacationDays:      usedVacationDays,
				TotalVacationDays:     30, // Стандартное значение по ТК РФ
				RemainingVacationDays: 30 - usedVacationDays,
			}, nil
		}
		return nil, err
	}

	// Преобразуем значение из строки в число
	var settingValue string
	if setting.SettingValue.Valid {
		settingValue = setting.SettingValue.String
	} else {
		// Если значение NULL, используем значение по умолчанию
		settingValue = "30"
	}

	totalVacationDays, err := strconv.ParseInt(settingValue, 10, 64)
	if err != nil {
		// Если не удалось преобразовать, используем значение по умолчанию
		totalVacationDays = 30
	}

	// Рассчитываем оставшиеся дни
	remainingVacationDays := max(totalVacationDays-usedVacationDays, 0)

	return &VacationStatisticsResponse{
		UsedVacationDays:      usedVacationDays,
		TotalVacationDays:     totalVacationDays,
		RemainingVacationDays: remainingVacationDays,
	}, nil
}

func (s *service) GetReportStatistics(ctx context.Context, userId string, month int, year int, gender int) (*ReportStatisticsResponse, error) {
	firstDayOfMonth := date.FirstDayOfMonth(month, year)

	var stat repo.GetMonthlyStatisticsRow
	var standard repo.WorkStandard
	var standardErr error

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		stat, err = s.repo.GetMonthlyStatistics(gCtx, repo.GetMonthlyStatisticsParams{
			UserID: userId,
			Year:   firstDayOfMonth,
			Month:  firstDayOfMonth,
		})
		return err
	})

	g.Go(func() error {
		var err error
		standard, err = s.getWorkStandardWithPriority(gCtx, userId, int32(month), int32(year), int32(gender))
		if err == sql.ErrNoRows {
			standardErr = err
			return nil
		}
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var standardHours int32
	var standardDays int32
	if standardErr == nil {
		standardHours = standard.StandardHours
		standardDays = standard.StandardDays
	}

	return &ReportStatisticsResponse{
		Hours: HoursStatisticResponse{
			TotalHours:    parser.InterfaceToFloat32(stat.TotalHours),
			StandardHours: standardHours,
		},
		WorkDays: WorkDaysStatisticResponse{
			TotalWorkDays:    stat.WorkDays,
			StandardWorkDays: standardDays,
		},
		VacationDays: CountDaysResponse{Count: stat.VacationDays},
		MedicalDays:  CountDaysResponse{Count: stat.MedicalDays},
		TimeOffDays:  CountDaysResponse{Count: stat.TimeOffDays},
		DecreeDays:   CountDaysResponse{Count: stat.DecreeDays},
	}, nil
}
