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
)

var ErrUnavailable = errors.New("кадровый справочник недоступен или вернул некорректные данные")
var ErrScope = errors.New("не подтверждены отдел и компания сотрудника")

type Config struct {
	BaseURL string
}

type DepartmentScopeResolver struct {
	config Config
	http   *http.Client
}

func NewDepartmentScopeResolver(config Config) *DepartmentScopeResolver {
	return &DepartmentScopeResolver{config: config, http: &http.Client{
		Timeout:       5 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

type employee struct {
	UserID     string     `json:"user_id"`
	Company    string     `json:"company"`
	Department string     `json:"department"`
	Employees  []employee `json:"employees"`
}

type scope struct{ company, department string }

// Members compares both company and department from the live employee directory.
func (r *DepartmentScopeResolver) Members(ctx context.Context, actorID, sessionToken string) (map[string]bool, error) {
	if r == nil {
		return nil, ErrUnavailable
	}
	if actorID == "" {
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
	byUser := map[string]scope{}
	invalid := map[string]bool{}
	add := func(e employee, groupDepartment string) {
		if e.UserID == "" {
			return
		}
		// Duplicate user IDs are ambiguous, even when their labels happen to match.
		if _, exists := byUser[e.UserID]; exists {
			invalid[e.UserID] = true
		}
		company := strings.TrimSpace(e.Company)
		department := strings.TrimSpace(e.Department)
		groupDepartment = strings.TrimSpace(groupDepartment)
		if department == "" {
			department = groupDepartment
		} else if groupDepartment != "" && department != groupDepartment {
			invalid[e.UserID] = true
		}
		if company == "" || department == "" {
			invalid[e.UserID] = true
		}
		byUser[e.UserID] = scope{company, department}
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
	actor, ok := byUser[actorID]
	if !ok || invalid[actorID] {
		return nil, ErrScope
	}
	members := map[string]bool{}
	for id, s := range byUser {
		if id != actorID && !invalid[id] && s == actor {
			members[id] = true
		}
	}
	return members, nil
}
