package permission

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

// Отключить после диагностики.
const DebugLogs = true

var checkSequence atomic.Uint64

type debugLoggerKey struct{}

type Client struct {
	host      string
	serviceID string
	http      *http.Client
}

type PermissionRequest struct {
	SessionToken string
	Service      string
	Entity       string
	Action       string
	UserId       *string
}

type PermissionResponse struct {
	IsAccess bool
	Message  string
	Code     int32
	UserId   string
}

func NewClient(host, serviceID string) (*Client, error) {
	u, err := url.Parse(host)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("AUTH_SERVICE_HOST must be an HTTP(S) URL")
	}
	if serviceID == "" {
		return nil, fmt.Errorf("AUTH_SERVICE_ID is required")
	}
	return &Client{
		host:      strings.TrimRight(host, "/"),
		serviceID: serviceID,
		http: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func (c *Client) get(ctx context.Context, path, token string, result any) error {
	logger, _ := ctx.Value(debugLoggerKey{}).(*slog.Logger)
	if logger == nil {
		logger = slog.Default()
	}
	started := time.Now()
	if DebugLogs {
		logger.Info("[permission-debug] HTTP request", "method", http.MethodGet, "url", c.host+path, "session_present", token != "")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.host+path, nil)
	if err != nil {
		return fmt.Errorf("build auth request: %w", err)
	}
	req.AddCookie(&http.Cookie{Name: "session", Value: token})
	resp, err := c.http.Do(req)
	if err != nil {
		if DebugLogs {
			logger.Info("[permission-debug] HTTP error", "url", req.URL.String(), "duration", time.Since(started), "error", err)
		}
		return fmt.Errorf("request auth service: %w", err)
	}
	defer resp.Body.Close()
	var body strings.Builder
	if DebugLogs {
		defer func() {
			text := strings.ReplaceAll(body.String(), token, "[REDACTED]")
			truncated := len(text) > 8192
			if truncated {
				text = text[:8192]
			}
			logger.Info("[permission-debug] HTTP response", "url", req.URL.String(), "status", resp.StatusCode,
				"duration", time.Since(started), "body", text, "body_truncated", truncated)
		}()
	}
	if resp.StatusCode != http.StatusOK {
		if DebugLogs {
			_, _ = io.Copy(&body, io.LimitReader(resp.Body, 8193))
		}
		return fmt.Errorf("auth service returned status %d", resp.StatusCode)
	}
	var reader io.Reader = io.LimitReader(resp.Body, 4<<20)
	if DebugLogs {
		reader = io.TeeReader(reader, &body)
	}
	if err := json.NewDecoder(reader).Decode(result); err != nil {
		return fmt.Errorf("decode auth response: %w", err)
	}
	return nil
}

func (c *Client) Validate(ctx context.Context, req *PermissionRequest) (response *PermissionResponse, err error) {
	if DebugLogs {
		target := ""
		if req.UserId != nil {
			target = *req.UserId
		}
		logger := slog.Default().With("check_id", checkSequence.Add(1))
		ctx = context.WithValue(ctx, debugLoggerKey{}, logger)
		logger.Info("[permission-debug] validate", "service_id", c.serviceID, "service", req.Service,
			"entity", req.Entity, "action", req.Action, "target_user_id", target, "session_present", req.SessionToken != "")
		defer func() {
			if err != nil {
				logger.Info("[permission-debug] validation failed", "error", err)
				return
			}
			logger.Info("[permission-debug] validation result", "allowed", response.IsAccess,
				"user_id", response.UserId, "code", response.Code, "message", response.Message)
		}()
	}
	if req.SessionToken == "" {
		return nil, fmt.Errorf("session is required")
	}
	var user struct {
		ID string `json:"id"`
	}
	if err := c.get(ctx, "/api/as/users/me", req.SessionToken, &user); err != nil {
		return nil, err
	}
	if user.ID == "" {
		return nil, fmt.Errorf("auth service returned an empty user ID")
	}
	var permissions []struct {
		Code string `json:"code"`
	}
	if err := c.get(ctx, "/api/as/users/me/permissions/"+url.PathEscape(c.serviceID), req.SessionToken, &permissions); err != nil {
		return nil, err
	}

	entity := req.Entity
	if req.UserId != nil && *req.UserId != "" && *req.UserId != user.ID {
		entity += ".all"
	}
	for _, p := range permissions {
		parts := strings.Split(p.Code, ":")
		if len(parts) == 3 &&
			(parts[0] == req.Service || parts[0] == "all") &&
			(parts[1] == entity || parts[1] == "all") &&
			(parts[2] == req.Action || parts[2] == "all") {
			return &PermissionResponse{IsAccess: true, UserId: user.ID}, nil
		}
	}
	return &PermissionResponse{Message: "Нет доступа к действию", Code: http.StatusForbidden, UserId: user.ID}, nil
}
