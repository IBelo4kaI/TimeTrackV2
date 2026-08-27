// Package timesheetreminder — напоминание сотруднику заполнить табель:
// мягкое в последние дни текущего месяца (за ещё не прошедшие дни, конечно,
// не спрашиваем) и более настойчивое (повторяется раз в день, пока не
// заполнено) в первые дни следующего месяца — про уже закрытый предыдущий.
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

	// softWindowDays — мягкое напоминание в последние N дней текущего месяца.
	softWindowDays = 3
	// hardWindowDays — настойчивое напоминание в первые N дней следующего
	// месяца, про предыдущий (уже закрытый) месяц.
	hardWindowDays = 7
	// runAtHour — час (UTC), в который тикер фактически выполняет проверку;
	// остальные тики в течение суток — no-op (см. Run).
	runAtHour = 6
)

var monthNames = [...]string{
	"январь", "февраль", "март", "апрель", "май", "июнь",
	"июль", "август", "сентябрь", "октябрь", "ноябрь", "декабрь",
}

// GapResult — один сотрудник с незаполненными днями за конкретный месяц.
// Notified — реально ли ушло уведомление именно сейчас (false, если раньше
// сегодня уже слали — см. CountNotificationsSentToday, — пропуски при этом
// у него всё равно есть и он всё равно попадает в список).
type GapResult struct {
	UserID   string `json:"userId"`
	Kind     string `json:"kind"` // "soft" | "hard"
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
// /timesheet-reminder/run), в обход не только часового/суточного гейта из
// Run, но и календарного окна (последние/первые дни месяца, см.
// runDailyCheck) — иначе в любой другой день ручной запуск молча ничего
// не делал бы, что бесполезно и для проверки, и как операционная кнопка
// "проверить прямо сейчас". Дедуп по notifications в БД
// (CountNotificationsSentToday) при этом никуда не девается: повторный
// запуск в тот же день для уже уведомлённого пользователя+месяца ничего
// не задублирует.
// RunNow — см. комментарий выше, дополнительно возвращает список тех, у
// кого нашлись пропуски (в т.ч. если уведомление сегодня уже уходило и
// сейчас подавлено дедупом — пропуски у человека всё равно есть).
func (s *Service) RunNow(ctx context.Context) []GapResult {
	now := time.Now().UTC()
	return s.checkAllUsers(ctx, now, true, true)
}

func (s *Service) runDailyCheck(ctx context.Context, now time.Time) {
	lastDayOfCurrentMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	inSoftWindow := now.Day() > lastDayOfCurrentMonth-softWindowDays
	inHardWindow := now.Day() <= hardWindowDays

	if !inSoftWindow && !inHardWindow {
		return
	}

	s.checkAllUsers(ctx, now, inSoftWindow, inHardWindow)
}

func (s *Service) checkAllUsers(ctx context.Context, now time.Time, checkSoft, checkHard bool) []GapResult {
	userIDs, err := s.repo.ListKnownUserIDs(ctx)
	if err != nil {
		s.logger.Error("timesheet reminder: list users failed", "err", err)
		return nil
	}

	var results []GapResult
	for _, userID := range userIDs {
		if checkSoft {
			if r, ok := s.checkAndNotify(ctx, userID, now.Year(), int(now.Month()), now, "soft"); ok {
				results = append(results, r)
			}
		}
		if checkHard {
			prevMonth := now.AddDate(0, -1, 0)
			if r, ok := s.checkAndNotify(ctx, userID, prevMonth.Year(), int(prevMonth.Month()), now, "hard"); ok {
				results = append(results, r)
			}
		}
	}
	return results
}

// checkAndNotify — считает пропуски в табеле пользователя за конкретный
// месяц; если они есть — шлёт напоминание (не чаще раза в день на
// пользователя+месяц+режим, см. CountNotificationsSentToday) и возвращает
// (GapResult, true) в любом случае, отправилось реально уведомление или
// подавлено дедупом (это в GapResult.Notified). now нужен только для
// "soft": ещё не наступившие дни текущего месяца не считаем.
func (s *Service) checkAndNotify(ctx context.Context, userID string, targetYear, targetMonth int, now time.Time, kind string) (GapResult, bool) {
	days, err := s.calendarService.GetCalendarDays(ctx, userID, targetMonth, targetYear)
	if err != nil {
		s.logger.Error("timesheet reminder: get calendar days failed", "err", err, "userId", userID)
		return GapResult{}, false
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	gaps := 0
	for _, day := range days.Days {
		if day.IsWeekend || day.UserTimeId != "" {
			continue
		}
		if kind == "soft" && !day.Date.Before(today) {
			continue
		}
		gaps++
	}
	if gaps == 0 {
		return GapResult{}, false
	}

	result := GapResult{UserID: userID, Kind: kind, Year: targetYear, Month: targetMonth, Gaps: gaps}

	entityID := fmt.Sprintf("%s:%04d-%02d", kind, targetYear, targetMonth)

	sentToday, err := s.repo.CountNotificationsSentToday(ctx, repo.CountNotificationsSentTodayParams{
		UserID:     userID,
		EntityType: sql.NullString{String: entityType, Valid: true},
		EntityID:   sql.NullString{String: entityID, Valid: true},
	})
	if err != nil {
		s.logger.Error("timesheet reminder: dedup check failed", "err", err, "userId", userID)
		return result, true
	}
	if sentToday > 0 {
		return result, true
	}

	title, body := buildText(kind, targetYear, targetMonth, gaps)

	s.notificationService.CreateMany(ctx, []string{userID}, title, body, repo.NotificationsTypeWarn, entityType, entityID)
	s.vkService.Notify(ctx, userID, title+": "+body, s.frontendURL+"/calendar")
	result.Notified = true
	return result, true
}

func buildText(kind string, year, month, gaps int) (title, body string) {
	monthName := monthNames[month-1]

	if kind == "soft" {
		return "Не забудьте заполнить табель",
			fmt.Sprintf("Незаполненных рабочих дней в этом месяце: %d", gaps)
	}
	return "В табеле остались незаполненные дни",
		fmt.Sprintf("За %s %d: незаполненных рабочих дней — %d", monthName, year, gaps)
}
