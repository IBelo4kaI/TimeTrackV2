package vacation

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v3"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/adapter/permission"
	"timetrack/internal/adapter/reference"
	"timetrack/internal/middleware"
	"timetrack/internal/response"
)

func (s *vacationService) DepartmentMembers(ctx context.Context, actorID, sessionToken string) (map[string]bool, error) {
	return s.departmentScope.Members(ctx, actorID, sessionToken)
}

func (s *vacationService) DepartmentDecision(ctx context.Context, id, decision, comment, sessionToken string) error {
	actor := actorFrom(ctx).ID
	comment = strings.TrimSpace(comment)
	if actor == "" || strings.TrimSpace(sessionToken) == "" {
		return fiber.ErrUnauthorized
	}
	if decision != "approved" && decision != "rejected" {
		return fiber.NewError(400, "Допустимые решения: approved, rejected")
	}
	if utf8.RuneCountInString(comment) > 4000 {
		return fiber.NewError(400, "Комментарий не должен превышать 4000 символов")
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
	if locked.UserID == actor || locked.UserID == "" {
		return fiber.NewError(403, "Самосогласование запрещено")
	}
	members, err := s.DepartmentMembers(ctx, actor, sessionToken)
	if err != nil {
		return departmentScopeError(err)
	}
	if !members[locked.UserID] {
		return fiber.NewError(403, "Нет доступа к заявке другого отдела или компании")
	}
	v, err := q.GetVacationByID(ctx, id)
	if err != nil {
		return err
	}
	if locked.Status != repo.VacationsStatusPending || v.DepartmentStatus != repo.VacationsDepartmentStatusPending {
		return workflowError("Этап согласования уже завершён или не требуется")
	}
	if err := q.SetDepartmentDecision(ctx, repo.SetDepartmentDecisionParams{ID: id, DepartmentStatus: repo.VacationsDepartmentStatus(decision), DepartmentActorID: actor, DepartmentComment: comment}); err != nil {
		return err
	}
	detail := "Руководитель отдела согласовал заявку"
	if decision == "rejected" {
		detail = "Руководитель отдела отклонил заявку"
	}
	if comment != "" {
		detail += ": " + comment
	}
	if err := q.AddVacationHistory(ctx, repo.AddVacationHistoryParams{VacationID: id, ActorID: actor, EventType: "department_" + decision, Detail: detail}); err != nil {
		return err
	}
	return tx.Commit()
}

func resetDepartmentDecision(ctx context.Context, q *repo.Queries, id, actor string) error {
	if err := q.ResetDepartmentDecision(ctx, id); err != nil {
		return err
	}
	return q.AddVacationHistory(ctx, repo.AddVacationHistoryParams{VacationID: id, ActorID: actor, EventType: "department_reset", Detail: "Согласование отдела сброшено после изменения или возврата заявки"})
}
func addDepartmentBypass(ctx context.Context, q *repo.Queries, id, actor, status string) error {
	return q.AddVacationHistory(ctx, repo.AddVacationHistoryParams{VacationID: id, ActorID: actor, EventType: "department_bypassed", Detail: fmt.Sprintf("Администратор утвердил заявку без согласования отдела (этап: %s)", status)})
}
func departmentScopeError(err error) error {
	if err == reference.ErrScope {
		return fiber.NewError(403, err.Error())
	}
	return fiber.NewError(503, err.Error())
}

func (h *Handler) hasDepartmentPermission(c fiber.Ctx) bool {
	actor, _ := c.Locals("user_id").(string)
	if actor == "" {
		return false
	}
	result, err := h.permission.Validate(c.RequestCtx(), &permission.PermissionRequest{SessionToken: c.Cookies(middleware.SessionCookieName), Service: h.prefix, Entity: "department.manager", Action: "manage"})
	return err == nil && result.IsAccess && result.UserId == actor
}
func (h *Handler) departmentMembers(c fiber.Ctx) (map[string]bool, error) {
	if !h.hasDepartmentPermission(c) {
		return nil, fiber.ErrForbidden
	}
	actor, _ := c.Locals("user_id").(string)
	members, err := h.service.DepartmentMembers(c.RequestCtx(), actor, c.Cookies(middleware.SessionCookieName))
	if err != nil {
		return nil, departmentScopeError(err)
	}
	return members, nil
}

// A manager can read a full application only after the handler checks its owner scope.
func (h *Handler) requireApplicationRead(c fiber.Ctx) error {
	token := c.Cookies(middleware.SessionCookieName)
	if token == "" {
		return fiber.ErrUnauthorized
	}
	for _, p := range []permission.PermissionRequest{
		{SessionToken: token, Service: h.prefix, Entity: "vacation", Action: "read"},
		{SessionToken: token, Service: h.prefix, Entity: "department.manager", Action: "manage"},
	} {
		result, err := h.permission.Validate(c.RequestCtx(), &p)
		if err == nil && result.IsAccess && result.UserId != "" {
			c.Locals("user_id", result.UserId)
			return c.Next()
		}
	}
	return fiber.ErrForbidden
}

type vacationResponse struct {
	repo.GetVacationByIDRow
	CanDepartmentApprove bool `json:"canDepartmentApprove"`
}
type userVacationResponse struct {
	repo.GetVacationsByYearRow
	CanDepartmentApprove bool `json:"canDepartmentApprove"`
}
type allVacationResponse struct {
	repo.GetAllUsersVacationsByYearRow
	CanDepartmentApprove bool `json:"canDepartmentApprove"`
}

func canDepartmentApprove(members map[string]bool, owner string, status repo.VacationsStatus, department repo.VacationsDepartmentStatus) bool {
	return members[owner] && status == repo.VacationsStatusPending && department == repo.VacationsDepartmentStatusPending
}
func (h *Handler) DepartmentDecision(c fiber.Ctx) error {
	var body struct {
		Decision string `json:"decision"`
		Comment  string `json:"comment"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}
	if err := h.service.DepartmentDecision(actionContext(c, ""), c.Params("id"), body.Decision, body.Comment, c.Cookies(middleware.SessionCookieName)); err != nil {
		return workflowResponseError(c, err)
	}
	return response.Success(c, fiber.Map{"message": "Решение руководителя отдела сохранено"})
}
func (h *Handler) DepartmentVacations(c fiber.Ctx) error {
	year, err := strconv.Atoi(c.Params("year"))
	if err != nil || year < 1 || year > 9999 {
		return response.BadRequest(c)
	}
	members, err := h.departmentMembers(c)
	if err != nil {
		return workflowResponseError(c, err)
	}
	rows, err := h.service.GetAllUserVacationsByYear(c.RequestCtx(), year)
	if err != nil {
		return workflowResponseError(c, err)
	}
	result := make([]allVacationResponse, 0)
	for _, v := range *rows {
		if members[v.UserID] {
			result = append(result, allVacationResponse{v, canDepartmentApprove(members, v.UserID, v.Status, v.DepartmentStatus)})
		}
	}
	c.Set("Cache-Control", "no-store")
	return response.Success(c, result)
}
