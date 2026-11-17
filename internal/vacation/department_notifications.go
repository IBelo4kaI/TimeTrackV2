package vacation

import (
	"context"
	"fmt"
	repo "timetrack/internal/adapter/mysql/sqlc"
)

func (s *vacationService) notifyDepartmentManagers(ctx context.Context, id, userID string, excluded []string) {
	session := actorFrom(ctx).Session
	if session == "" {
		return
	}
	vacation, err := s.repo.GetVacationByID(ctx, id)
	if err != nil || vacation.Status != repo.VacationsStatusPending || vacation.DepartmentStatus != repo.VacationsDepartmentStatusPending {
		return
	}
	directory, err := s.departmentScope.Directory(ctx, session)
	if err != nil {
		return
	}
	department, ok := directory[userID]
	if !ok {
		return
	}
	assignments, err := s.repo.ListDepartmentManagersByScope(ctx, repo.ListDepartmentManagersByScopeParams{Company: department.Company, Department: department.Name})
	if err != nil {
		return
	}
	seen := map[string]bool{userID: true}
	for _, id := range excluded {
		seen[id] = true
	}
	recipients := make([]string, 0, len(assignments))
	for _, assignment := range assignments {
		current, ok := directory[assignment.UserID]
		if !ok || current != department || seen[assignment.UserID] {
			continue
		}
		seen[assignment.UserID] = true
		recipients = append(recipients, assignment.UserID)
	}
	if len(recipients) == 0 {
		return
	}
	title := "Заявка на согласование отдела"
	dates := fmt.Sprintf("%s – %s", vacation.StartDate.Format("02.01.2006"), vacation.EndDate.Format("02.01.2006"))
	body := vacationNotificationBody(s.resolveApplicantName(ctx, userID), dates, vacation.VacationTypeName, "")
	s.notificationService.CreateMany(ctx, recipients, title, body, repo.NotificationsTypeInfo, "vacation", id)
	s.vkService.NotifyMany(ctx, recipients, title+": "+body, fmt.Sprintf("%s/docs/vacation/%s", s.frontendURL, id))
}
