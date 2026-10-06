// Package timesheetreminder — напоминание сотруднику заполнить табель за
// уже закрытый предыдущий месяц: одно на сотрудника и месяц, в первые дни
// следующего (окно из нескольких дней нужно только на случай, если сервер в
// первый день был недоступен).
// Списка "все сотрудники" в проекте нет (в auth-сервис за ним намеренно не
// ходим, см. cmd/api.go) — проверяем только тех, о ком бэк и так уже что-то
// знает локально (см. ListKnownUserIDs): кто хоть раз вносил запись в
// табель или кому задан индивидуальный график. Сотрудник, который вообще
// ничего не вносил, под проверку не попадёт.
package timesheetreminder

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/calendar"
	"timetrack/internal/notification"
	"timetrack/internal/vk"
)

const (
	entityType = "timesheet"

	// reminderWindowDays — напоминание в первые N дней месяца про предыдущий
	// (уже закрытый); слать его нужно один раз, окно — страховка на сбои.
	reminderWindowDays = 7
	// runAtHour — час (UTC), в который тикер фактически выполняет проверку;
	// остальные тики в течение суток — no-op (см. Run).
	runAtHour = 6
)

var monthNames = [...]string{
	"январь", "февраль", "март", "апрель", "май", "июнь",
	"июль", "август", "сентябрь", "октябрь", "ноябрь", "декабрь",
}

// GapResult — один сотрудник с незаполненными днями за конкретный месяц.
// Notified — реально ли ушло уведомление именно сейчас (false, если за этот
// месяц его уже слали — см. CountNotificationsByEntity, — пропуски при этом
// у него всё равно есть и он всё равно попадает в список).
type GapResult struct {
	UserID   string `json:"userId"`
	Year     int    `json:"year"`
	Month    int    `json:"month"`
	Gaps     int    `json:"gaps"`
	Notified bool   `json:"notified"`
}

type Service struct {
	repo                repo.Querier
	calendarService     calendar.Service
	notificationService notification.Service
	vkService           vk.Service
	frontendURL         string
	logger              *slog.Logger

	lastRunDate string // "2026-08-27" — чтобы не запускать дважды за сутки
}

func NewService(
	r repo.Querier,
	calendarService calendar.Service,
	notificationService notification.Service,
	vkService vk.Service,
	frontendURL string,
	logger *slog.Logger,
) *Service {
	return &Service{
		repo:                r,
		calendarService:     calendarService,
		notificationService: notificationService,
		vkService:           vkService,
		frontendURL:         frontendURL,
		logger:              logger,
	}
}

// Run — вызывать раз в час (см. cmd/api.go): фактическая проверка идёт раз
// в сутки, в runAtHour, остальные вызовы — no-op. Так переживает рестарт
// сервера в произвольный момент, не завязываясь на время старта процесса.
func (s *Service) Run(ctx context.Context, now time.Time) {
	if now.Hour() != runAtHour {
		return
	}
	today := now.Format("2006-01-02")
	if s.lastRunDate == today {
		return
	}
	s.lastRunDate = today

	s.runDailyCheck(ctx, now)
}

// RunNow — принудительный прогон прямо сейчас (см. handler.go — POST
// /timesheet-reminder/run), в обход часового/суточного гейта из Run и
// календарного окна (первые дни месяца) — иначе в любой другой день ручной
// запуск молча ничего не делал бы. Дедуп по notifications в БД при этом
// никуда не девается: за один месяц сотрудник уведомляется один раз.
// Возвращает всех, у кого нашлись пропуски (в т.ч. если уведомление уже
// уходило и сейчас подавлено дедупом).
func (s *Service) RunNow(ctx context.Context) []GapResult {
	return s.checkAllUsers(ctx, time.Now().UTC())
}

func (s *Service) runDailyCheck(ctx context.Context, now time.Time) {
	if now.Day() > reminderWindowDays {
		return
	}

	s.checkAllUsers(ctx, now)
}

func (s *Service) checkAllUsers(ctx context.Context, now time.Time) []GapResult {
	userIDs, err := s.repo.ListKnownUserIDs(ctx)
	if err != nil {
		s.logger.Error("timesheet reminder: list users failed", "err", err)
		return nil
	}

	prevMonth := now.AddDate(0, -1, 0)

	var results []GapResult
	for _, userID := range userIDs {
		if r, ok := s.checkAndNotify(ctx, userID, prevMonth.Year(), int(prevMonth.Month())); ok {
			results = append(results, r)
		}
	}
	return results
}

// checkAndNotify — считает пропуски в табеле пользователя за конкретный
// (закрытый) месяц; если они есть — шлёт напоминание, но один раз на
// пользователя и месяц (см. CountNotificationsByEntity), и возвращает
// (GapResult, true) в любом случае, отправилось реально уведомление или
// подавлено дедупом (это в GapResult.Notified).
func (s *Service) checkAndNotify(ctx context.Context, userID string, targetYear, targetMonth int) (GapResult, bool) {
	days, err := s.calendarService.GetCalendarDays(ctx, userID, targetMonth, targetYear)
	if err != nil {
		s.logger.Error("timesheet reminder: get calendar days failed", "err", err, "userId", userID)
		return GapResult{}, false
	}

	gaps := 0
	for _, day := range days.Days {
		if day.IsWeekend || day.UserTimeId != "" {
			continue
		}
		gaps++
	}
	if gaps == 0 {
		return GapResult{}, false
	}

	result := GapResult{UserID: userID, Year: targetYear, Month: targetMonth, Gaps: gaps}

	// Формат ключа прежний ("hard:"), чтобы не слать повторно за месяцы, по
	// которым напоминание уже ушло до перехода на одно напоминание
	entityID := fmt.Sprintf("hard:%04d-%02d", targetYear, targetMonth)

	sent, err := s.repo.CountNotificationsByEntity(ctx, repo.CountNotificationsByEntityParams{
		UserID:     userID,
		EntityType: sql.NullString{String: entityType, Valid: true},
		EntityID:   sql.NullString{String: entityID, Valid: true},
	})
	if err != nil {
		s.logger.Error("timesheet reminder: dedup check failed", "err", err, "userId", userID)
		return result, true
	}
	if sent > 0 {
		return result, true
	}

	title, body := buildText(targetYear, targetMonth, gaps)

	s.notificationService.CreateMany(ctx, []string{userID}, title, body, repo.NotificationsTypeWarn, entityType, entityID)
	s.vkService.Notify(ctx, userID, title+": "+body, s.frontendURL+"/calendar")
	result.Notified = true
	return result, true
}

func buildText(year, month, gaps int) (title, body string) {
	return "В табеле остались незаполненные дни",
		fmt.Sprintf("За %s %d: незаполненных рабочих дней — %d", monthNames[month-1], year, gaps)
}
