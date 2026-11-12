package permission

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.host+path, nil)
	if err != nil {
		return fmt.Errorf("build auth request: %w", err)
	}
	req.AddCookie(&http.Cookie{Name: "session", Value: token})
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request auth service: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth service returned status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(result); err != nil {
		return fmt.Errorf("decode auth response: %w", err)
	}
	return nil
}

func (c *Client) Validate(ctx context.Context, req *PermissionRequest) (*PermissionResponse, error) {
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
