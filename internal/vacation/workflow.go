package vacation

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v3"
	repo "timetrack/internal/adapter/mysql/sqlc"
)

type workflowKey struct{}
type workflowActor struct{ ID, Comment string }

func actionContext(c fiber.Ctx, comment string) context.Context {
	id, _ := c.Locals("user_id").(string)
	return context.WithValue(c.RequestCtx(), workflowKey{}, workflowActor{ID: id, Comment: strings.TrimSpace(comment)})
}
func actorFrom(ctx context.Context) workflowActor {
	a, _ := ctx.Value(workflowKey{}).(workflowActor)
	return a
}
func workflowError(message string) error { return fiber.NewError(fiber.StatusConflict, message) }
func validateTransition(from, to repo.VacationsStatus, comment string) error {
	if utf8.RuneCountInString(comment) > 4000 {
		return fiber.NewError(400, "Комментарий не должен превышать 4000 символов")
	}
	switch to {
	case repo.VacationsStatusApproved, repo.VacationsStatusRejected, repo.VacationsStatusWithdrawn:
		if from != repo.VacationsStatusPending {
			return workflowError("Действие доступно только для заявки на рассмотрении")
		}
	case repo.VacationsStatusPending:
		if from == repo.VacationsStatusPending {
			return workflowError("Заявка уже на рассмотрении")
		}
	default:
		return fiber.NewError(400, "Некорректный статус")
	}
	return nil
}

func (s *vacationService) transition(ctx context.Context, id string, status repo.VacationsStatus) error {
	actor := actorFrom(ctx)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.repo.WithTx(tx)
	locked, err := q.LockVacation(ctx, id)
	if err != nil {
		return err
	}
	if err := validateTransition(locked.Status, status, actor.Comment); err != nil {
		return err
	}
	vacation, err := q.GetVacationByID(ctx, id)
	if err != nil {
		return err
	}
	if status == repo.VacationsStatusWithdrawn && actor.ID != vacation.UserID {
		return fiber.NewError(403, "Отозвать заявку может только её автор")
	}
	// Существующий механизм синхронизации табеля сохраняется при решении и возврате.
	if status == repo.VacationsStatusApproved {
		if err := s.createOrUpdateVacationTimeEntries(ctx, vacation.UserID, vacation.StartDate, vacation.EndDate, q); err != nil {
			return err
		}
	} else if vacation.Status == repo.VacationsStatusApproved {
		if err := s.deleteVacationTimeEntries(ctx, vacation.UserID, vacation.StartDate, vacation.EndDate, q); err != nil {
			return err
		}
	}
	if err := q.SetVacationDecision(ctx, repo.SetVacationDecisionParams{Status: status, DecisionComment: actor.Comment, ID: id}); err != nil {
		return err
	}
	detail := map[repo.VacationsStatus]string{repo.VacationsStatusApproved: "Заявка утверждена", repo.VacationsStatusRejected: "Заявка отклонена", repo.VacationsStatusWithdrawn: "Заявка отозвана", repo.VacationsStatusPending: "Заявка возвращена на рассмотрение"}[status]
	if actor.Comment != "" {
		detail += ": " + actor.Comment
	}
	if err := q.AddVacationHistory(ctx, repo.AddVacationHistoryParams{VacationID: id, ActorID: actor.ID, EventType: string(status), Detail: detail}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.notifyApplicantStatusChanged(ctx, vacation, status, s.resolveApplicantName(ctx, vacation.UserID))
	if status == repo.VacationsStatusApproved {
		if err := s.SendApprovalEmailIfReady(ctx, id); err != nil {
			fmt.Printf("vacation: send approval email failed: %v\n", err)
		}
	}
	return nil
}

func (s *vacationService) GetHistory(ctx context.Context, id string) ([]repo.VacationHistory, error) {
	return s.repo.ListVacationHistory(ctx, id)
}

func (s *vacationService) EditApplication(ctx context.Context, id string, body VacationCreateRequest) error {
	if body.StartDate.IsZero() || body.EndDate.IsZero() || body.StartDate.After(body.EndDate) {
		return fiber.NewError(400, "Укажите корректный период отпуска")
	}
	calculated, err := s.CalculateVacationDays(ctx, body.StartDate, body.EndDate)
	if err != nil {
		return err
	}
	if calculated.TotalVacationDays < 1 {
		return fiber.NewError(400, "В периоде должен быть хотя бы один день отпуска")
	}
	typeID, err := s.resolveVacationTypeID(ctx, body.VacationTypeID)
	if err != nil {
		return err
	}
	affects := true
	typeName := "Основной оплачиваемый"
	if typeID.Valid {
		t, err := s.repo.GetVacationTypeByID(ctx, typeID.String)
		if err != nil {
			return err
		}
		affects = t.AffectsBalance
		typeName = t.Name
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.repo.WithTx(tx)
	locked, err := q.LockVacation(ctx, id)
	if err != nil {
		return err
	}
	if locked.Status != repo.VacationsStatusPending {
		return workflowError("Редактировать можно только заявку на рассмотрении")
	}
	previous, err := q.GetVacationByID(ctx, id)
	if err != nil {
		return err
	}
	stats, err := s.GetVacationsStats(ctx, locked.UserID, body.StartDate.Year())
	if err != nil {
		return err
	}
	excess := 0
	if affects {
		excess = calculateExcessDays(calculated.TotalVacationDays, stats.Free)
	}
	if err := q.EditPendingVacation(ctx, repo.EditPendingVacationParams{ID: id, StartDate: body.StartDate, EndDate: body.EndDate, TotalDays: int32(calculated.TotalVacationDays), VacationTypeID: typeID, Description: sql.NullString{String: body.Description, Valid: body.Description != ""}, ExcessDays: int32(excess), AvailableDays: sql.NullInt32{Int32: int32(stats.Free), Valid: true}}); err != nil {
		return err
	}
	eventType := "edited"
	detail := fmt.Sprintf("Заявка изменена: %s — %s, %s", body.StartDate.Format("02.01.2006"), body.EndDate.Format("02.01.2006"), typeName)
	if previous.VacationTypeID != typeID && previous.StartDate.Format("2006-01-02") == body.StartDate.Format("2006-01-02") && previous.EndDate.Format("2006-01-02") == body.EndDate.Format("2006-01-02") && previous.Description == body.Description {
		eventType = "type_changed"
		detail = "Изменён тип отпуска: " + previous.VacationTypeName + " → " + typeName
	}
	if err := q.AddVacationHistory(ctx, repo.AddVacationHistoryParams{VacationID: id, ActorID: actorFrom(ctx).ID, EventType: eventType, Detail: detail}); err != nil {
		return err
	}
	return tx.Commit()
}

// MySQL legacy zero dates (or timestamps shifted into year 0 by a timezone)
// cannot be encoded by time.Time.MarshalJSON. Preserve the event, not a made-up date.
func historyResponse(events []repo.VacationHistory) []HistoryEvent {
	result := make([]HistoryEvent, 0, len(events))
	for _, event := range events {
		var createdAt *time.Time
		if !event.CreatedAt.IsZero() && event.CreatedAt.Year() >= 1 && event.CreatedAt.Year() <= 9999 {
			if _, err := event.CreatedAt.MarshalJSON(); err == nil {
				stamp := event.CreatedAt
				createdAt = &stamp
			}
		}
		result = append(result, HistoryEvent{ID: event.ID, VacationID: event.VacationID, ActorID: event.ActorID, EventType: event.EventType, Detail: event.Detail, CreatedAt: createdAt})
	}
	return result
}
