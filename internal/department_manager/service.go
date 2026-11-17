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
	EnsureDepartmentManagerRole(ctx context.Context, userID string) error
}

type Service struct {
	repo      *repo.Queries
	directory *reference.DepartmentScopeResolver
	roles     RoleAssigner
}

func NewService(queries *repo.Queries, directory *reference.DepartmentScopeResolver, roles RoleAssigner) *Service {
	return &Service{repo: queries, directory: directory, roles: roles}
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
	assignment, err := s.repo.GetDepartmentManager(ctx, id)
	if err != nil {
		return repo.DepartmentManager{}, err
	}
	status := repo.DepartmentManagersRoleSyncStatusNotConfigured
	if s.roles != nil {
		status = repo.DepartmentManagersRoleSyncStatusSynced
		if err := s.roles.EnsureDepartmentManagerRole(ctx, assignment.UserID); err != nil {
			status = repo.DepartmentManagersRoleSyncStatusFailed
		}
	}
	if err := s.repo.SetDepartmentManagerRoleSync(ctx, repo.SetDepartmentManagerRoleSyncParams{ID: id, RoleSyncStatus: status}); err != nil {
		return repo.DepartmentManager{}, err
	}
	return s.repo.GetDepartmentManager(ctx, id)
}
func (s *Service) Delete(ctx context.Context, id string) error {
	count, err := s.repo.DeleteDepartmentManager(ctx, id)
	if err == nil && count == 0 {
		return sql.ErrNoRows
	}
	return err
}
