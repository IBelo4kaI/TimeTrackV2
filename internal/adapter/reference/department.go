package reference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
)

var ErrUnavailable = errors.New("кадровый справочник недоступен или вернул некорректные данные")
var ErrScope = errors.New("не подтверждён отдел сотрудника")

type Config struct {
	BaseURL string
}

type DepartmentScopeResolver struct {
	config   Config
	managers *repo.Queries
	http     *http.Client
}

func NewDepartmentScopeResolver(config Config, managers *repo.Queries) *DepartmentScopeResolver {
	return &DepartmentScopeResolver{config: config, managers: managers, http: &http.Client{
		Timeout:       5 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

type employee struct {
	UserID     string     `json:"user_id"`
	FullName   any        `json:"full_name"`
	Company    any        `json:"company"`
	Department string     `json:"department"`
	Employees  []employee `json:"employees"`
}

type Department struct {
	Company string `json:"company"`
	Name    string `json:"department"`
}

// Members resolves saved target scopes against the live employee directory.
func (r *DepartmentScopeResolver) Members(ctx context.Context, actorID, sessionToken string) (map[string]bool, error) {
	if r == nil || r.managers == nil {
		return nil, ErrUnavailable
	}
	if strings.TrimSpace(actorID) == "" {
		return nil, ErrScope
	}
	assignments, err := r.managers.ListDepartmentManagersByUser(ctx, actorID)
	if err != nil {
		return nil, ErrUnavailable
	}
	if len(assignments) == 0 {
		return nil, ErrScope
	}
	scopes := map[string]bool{}
	for _, assignment := range assignments {
		if strings.TrimSpace(assignment.Department) == "" {
			continue
		}
		scopes[assignment.Department] = true
	}
	if len(scopes) == 0 {
		return nil, ErrScope
	}
	directory, err := r.Directory(ctx, sessionToken)
	if err != nil {
		return nil, err
	}
	if _, ok := directory[actorID]; !ok {
		return nil, ErrScope
	}
	members := map[string]bool{}
	for id, department := range directory {
		if id != actorID && scopes[department.Name] {
			members[id] = true
		}
	}
	return members, nil
}

type DirectoryEmployee struct {
	Department
	FullName string
}

func (r *DepartmentScopeResolver) Directory(ctx context.Context, sessionToken string) (map[string]Department, error) {
	employees, err := r.Employees(ctx, sessionToken)
	if err != nil {
		return nil, err
	}
	directory := make(map[string]Department, len(employees))
	for id, employee := range employees {
		directory[id] = employee.Department
	}
	return directory, nil
}

func (r *DepartmentScopeResolver) Employees(ctx context.Context, sessionToken string) (map[string]DirectoryEmployee, error) {
	if r == nil {
		return nil, ErrUnavailable
	}
	cfg := r.config
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, ErrUnavailable
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(cfg.BaseURL, "/")+"/apiref/ref/employees/internal", nil)
	if err != nil {
		return nil, ErrUnavailable
	}
	if strings.TrimSpace(sessionToken) == "" {
		return nil, ErrUnavailable
	}
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionToken})
	resp, err := r.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: ошибка запроса", ErrUnavailable)
	}
	defer resp.Body.Close()
	const limit = 4 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: HTTP %d", ErrUnavailable, resp.StatusCode)
	}
	if err != nil {
		return nil, ErrUnavailable
	}
	if len(data) > limit {
		return nil, ErrUnavailable
	}
	var records []employee
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("%w: неверный формат ответа", ErrUnavailable)
	}
	byUser := map[string]DirectoryEmployee{}
	invalid := map[string]bool{}
	add := func(e employee, groupDepartment string) {
		if e.UserID == "" {
			return
		}
		// Duplicate user IDs are ambiguous, even when their labels happen to match.
		if _, exists := byUser[e.UserID]; exists {
			invalid[e.UserID] = true
		}
		company, _ := e.Company.(string)
		company = strings.TrimSpace(company)
		department := strings.TrimSpace(e.Department)
		groupDepartment = strings.TrimSpace(groupDepartment)
		if department == "" {
			department = groupDepartment
		} else if groupDepartment != "" && department != groupDepartment {
			invalid[e.UserID] = true
		}
		if department == "" {
			invalid[e.UserID] = true
		}
		fullName, _ := e.FullName.(string)
		byUser[e.UserID] = DirectoryEmployee{Department: Department{Company: company, Name: department}, FullName: strings.TrimSpace(fullName)}
	}
	for _, record := range records {
		if record.Employees != nil {
			if record.UserID != "" {
				return nil, ErrUnavailable
			}
			for _, e := range record.Employees {
				if e.Employees != nil {
					return nil, ErrUnavailable
				}
				add(e, record.Department)
			}
		} else {
			add(record, "")
		}
	}
	for id := range invalid {
		delete(byUser, id)
	}
	return byUser, nil
}
