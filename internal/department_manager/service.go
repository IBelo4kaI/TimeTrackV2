package departmentmanager

import (
	"context"
	"database/sql"
	"errors"
	mysql "github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"sort"
	"strings"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/adapter/reference"
	"unicode/utf8"
)

// RoleAssigner must ensure the role idempotently without replacing other user roles.
type RoleAssigner interface {
	EnsureRole(ctx context.Context, userID, roleID string) error
	RemoveRole(ctx context.Context, userID, roleID string) error
}

type Service struct {
	db        *sql.DB
	repo      *repo.Queries
	directory *reference.DepartmentScopeResolver
	roles     RoleAssigner
}

func NewService(queries *repo.Queries, db *sql.DB, directory *reference.DepartmentScopeResolver, roles RoleAssigner) *Service {
	return &Service{repo: queries, db: db, directory: directory, roles: roles}
}

type AssignmentRequest struct {
	UserID     string `json:"userId"`
	Company    string `json:"company"`
	Department string `json:"department"`
}
type Candidate struct {
	UserID     string `json:"userId"`
	Company    string `json:"company"`
	Department string `json:"department"`
}

func (s *Service) List(ctx context.Context) ([]repo.DepartmentManager, error) {
	rows, err := s.repo.ListDepartmentManagers(ctx)
	if rows == nil {
		rows = []repo.DepartmentManager{}
	}
	return rows, err
}
func (s *Service) Candidates(ctx context.Context, session string) ([]Candidate, error) {
	directory, err := s.directory.Directory(ctx, session)
	if err != nil {
		return nil, fiber.NewError(503, "Не удалось получить однозначные данные сотрудников из справочника")
	}
	result := make([]Candidate, 0, len(directory))
	for id, dept := range directory {
		result = append(result, Candidate{UserID: id, Company: dept.Company, Department: dept.Name})
	}
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.Company != b.Company {
			return a.Company < b.Company
		}
		if a.Department != b.Department {
			return a.Department < b.Department
		}
		return a.UserID < b.UserID
	})
	return result, nil
}
func (s *Service) Create(ctx context.Context, body AssignmentRequest, actorID, session string) (repo.DepartmentManager, error) {
	body.Company = strings.TrimSpace(body.Company)
	body.Department = strings.TrimSpace(body.Department)
	if _, err := uuid.Parse(body.UserID); err != nil || body.Company == "" || body.Department == "" || utf8.RuneCountInString(body.Company) > 255 || utf8.RuneCountInString(body.Department) > 255 {
		return repo.DepartmentManager{}, fiber.NewError(400, "Укажите сотрудника, компанию и отдел")
	}
	directory, err := s.directory.Directory(ctx, session)
	if err != nil {
		return repo.DepartmentManager{}, fiber.NewError(503, "Кадровый справочник недоступен")
	}
	dept, ok := directory[body.UserID]
	if !ok || dept.Company != body.Company || dept.Name != body.Department {
		return repo.DepartmentManager{}, fiber.NewError(409, "Компания или отдел сотрудника не подтверждены справочником")
	}
	id := uuid.NewString()
	err = s.repo.CreateDepartmentManager(ctx, repo.CreateDepartmentManagerParams{ID: id, Company: body.Company, Department: body.Department, UserID: body.UserID, CreatedBy: actorID})
	if err != nil {
		var duplicate *mysql.MySQLError
		if errors.As(err, &duplicate) && duplicate.Number == 1062 {
			return repo.DepartmentManager{}, fiber.NewError(409, "Этот сотрудник уже назначен руководителем отдела")
		}
		return repo.DepartmentManager{}, err
	}
	return s.SyncRole(ctx, id)
}
func (s *Service) SyncRole(ctx context.Context, id string) (repo.DepartmentManager, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return repo.DepartmentManager{}, err
	}
	defer tx.Rollback()
	q := s.repo.WithTx(tx)
	// Serialize role changes and settings updates before locking the assignment.
	setting, err := q.LockDepartmentManagerRoleSetting(ctx)
	if err != nil {
		return repo.DepartmentManager{}, err
	}
	assignment, err := q.LockDepartmentManager(ctx, id)
	if err != nil {
		return repo.DepartmentManager{}, err
	}
	desired := strings.TrimSpace(setting.String)
	current := assignment.AuthRoleID
	status := repo.DepartmentManagersRoleSyncStatusNotConfigured
	if s.roles != nil {
		status = repo.DepartmentManagersRoleSyncStatusSynced
		if current != "" && current != desired {
			if err := s.removeRoleIfUnused(ctx, q, assignment); err != nil {
				status = repo.DepartmentManagersRoleSyncStatusFailed
			} else {
				current = ""
			}
		}
		if status != repo.DepartmentManagersRoleSyncStatusFailed {
			if desired == "" {
				status = repo.DepartmentManagersRoleSyncStatusNotConfigured
			} else {
				if err := s.roles.EnsureRole(ctx, assignment.UserID, desired); err != nil {
					status = repo.DepartmentManagersRoleSyncStatusFailed
				} else {
					current = desired
				}
			}
		}
	}
	if err := q.UpdateDepartmentManagerRole(ctx, repo.UpdateDepartmentManagerRoleParams{ID: id, RoleSyncStatus: status, AuthRoleID: current}); err != nil {
		return repo.DepartmentManager{}, err
	}
	if err := tx.Commit(); err != nil {
		return repo.DepartmentManager{}, err
	}
	return s.repo.GetDepartmentManager(ctx, id)
}
func (s *Service) removeRoleIfUnused(ctx context.Context, q *repo.Queries, assignment repo.DepartmentManager) error {
	if assignment.AuthRoleID == "" {
		return nil
	}
	if s.roles == nil {
		return fiber.NewError(503, "Сервис назначения ролей недоступен")
	}
	count, err := q.CountOtherDepartmentManagersWithRole(ctx, repo.CountOtherDepartmentManagersWithRoleParams{UserID: assignment.UserID, AuthRoleID: assignment.AuthRoleID, ID: assignment.ID})
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return s.roles.RemoveRole(ctx, assignment.UserID, assignment.AuthRoleID)
}
func (s *Service) Delete(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.repo.WithTx(tx)
	if _, err := q.LockDepartmentManagerRoleSetting(ctx); err != nil {
		return err
	}
	assignment, err := q.LockDepartmentManager(ctx, id)
	if err != nil {
		return err
	}
	if err := s.removeRoleIfUnused(ctx, q, assignment); err != nil {
		if updateErr := q.SetDepartmentManagerRoleSync(ctx, repo.SetDepartmentManagerRoleSyncParams{ID: id, RoleSyncStatus: repo.DepartmentManagersRoleSyncStatusFailed}); updateErr != nil {
			return updateErr
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return commitErr
		}
		return fiber.NewError(503, "Не удалось снять роль в сервисе авторизации. Назначение сохранено; повторите удаление")
	}
	if _, err := q.DeleteDepartmentManager(ctx, id); err != nil {
		return err
	}
	return tx.Commit()
}
