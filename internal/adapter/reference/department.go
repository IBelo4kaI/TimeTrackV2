package reference

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrUnavailable = errors.New("кадровый справочник недоступен или вернул некорректные данные")
var ErrScope = errors.New("не подтверждены отдел и компания сотрудника")

type Config struct {
	BaseURL string
	APIKey  string
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
	UserID       string     `json:"user_id"`
	CompanyID    string     `json:"company_id"`
	DepartmentID string     `json:"department_id"`
	Employees    []employee `json:"employees"`
}

type scope struct{ company, department string }

// Members returns only unambiguous employees with explicit company and department IDs.
func (r *DepartmentScopeResolver) Members(ctx context.Context, actorID string) (map[string]bool, error) {
	if r == nil {
		log.Print("reference: scope lookup blocked: resolver is not configured")
		return nil, ErrUnavailable
	}
	if actorID == "" {
		log.Print("reference: scope lookup blocked: actor ID is missing")
		return nil, ErrUnavailable
	}
	cfg := r.config
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		log.Print("reference: scope lookup blocked: invalid APIREF_BASE_URL")
		return nil, ErrUnavailable
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(cfg.BaseURL, "/")+"/ref/employees/internal", nil)
	if err != nil {
		log.Print("reference: failed to build GET /ref/employees/internal request")
		return nil, ErrUnavailable
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		log.Print("reference: scope lookup blocked: AUTH_SERVICE_API_KEY is empty")
		return nil, ErrUnavailable
	}
	req.AddCookie(&http.Cookie{Name: "session", Value: cfg.APIKey})
	started := time.Now()
	log.Print("reference: GET /ref/employees/internal started")
	resp, err := r.http.Do(req)
	if err != nil {
		log.Printf("reference: GET /ref/employees/internal failed: elapsed=%s error_type=%T timeout=%t canceled=%t", time.Since(started), err, errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled))
		return nil, fmt.Errorf("%w: ошибка запроса", ErrUnavailable)
	}
	defer resp.Body.Close()
	log.Printf("reference: GET /ref/employees/internal response: status=%d content_type=%q elapsed=%s", resp.StatusCode, resp.Header.Get("Content-Type"), time.Since(started))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: HTTP %d", ErrUnavailable, resp.StatusCode)
	}
	const limit = 4 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		log.Printf("reference: response body read failed: error_type=%T", err)
		return nil, ErrUnavailable
	}
	if len(data) > limit {
		log.Printf("reference: response body exceeds limit: limit_bytes=%d", limit)
		return nil, ErrUnavailable
	}
	var records []employee
	if err := json.Unmarshal(data, &records); err != nil {
		bodyKind := "empty"
		if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 {
			switch trimmed[0] {
			case '<':
				bodyKind = "html_or_xml"
			case '[':
				bodyKind = "json_array"
			case '{':
				bodyKind = "json_object"
			default:
				bodyKind = "other"
			}
		}
		var syntaxError *json.SyntaxError
		var offset int64
		if errors.As(err, &syntaxError) {
			offset = syntaxError.Offset
		}
		log.Printf("reference: invalid employees response: bytes=%d error_type=%T body_kind=%s syntax_offset=%d", len(data), err, bodyKind, offset)
		return nil, fmt.Errorf("%w: неверный формат ответа", ErrUnavailable)
	}
	byUser := map[string]scope{}
	invalid := map[string]bool{}
	add := func(e employee) {
		if e.UserID == "" {
			return
		}
		// Duplicate user IDs are ambiguous, even when their labels happen to match.
		if _, exists := byUser[e.UserID]; exists {
			invalid[e.UserID] = true
		}
		if strings.TrimSpace(e.CompanyID) == "" || strings.TrimSpace(e.DepartmentID) == "" {
			invalid[e.UserID] = true
		}
		byUser[e.UserID] = scope{e.CompanyID, e.DepartmentID}
	}
	for _, record := range records {
		if record.Employees != nil {
			if record.UserID != "" {
				log.Print("reference: invalid employees response: group contains user_id")
				return nil, ErrUnavailable
			}
			for _, e := range record.Employees {
				if e.Employees != nil {
					log.Print("reference: invalid employees response: nested employee groups")
					return nil, ErrUnavailable
				}
				add(e)
			}
		} else {
			add(record)
		}
	}
	actor, ok := byUser[actorID]
	if !ok || invalid[actorID] {
		log.Printf("reference: scope lookup denied: actor_found=%t actor_scope_invalid=%t", ok, invalid[actorID])
		return nil, ErrScope
	}
	members := map[string]bool{}
	for id, s := range byUser {
		if id != actorID && !invalid[id] && s == actor {
			members[id] = true
		}
	}
	log.Printf("reference: scope lookup completed: employees=%d invalid_employees=%d department_members=%d elapsed=%s", len(byUser), len(invalid), len(members), time.Since(started))
	return members, nil
}
