package authservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (s *service) userHasRole(ctx context.Context, userID, roleID string) (bool, error) {
	users, err := s.GetAllUsers(ctx)
	if err != nil {
		return false, err
	}
	for _, user := range users {
		if user.ID != userID {
			continue
		}
		for _, role := range user.Roles {
			if role.ID == roleID {
				return true, nil
			}
		}
		return false, nil
	}
	return false, fmt.Errorf("user is not available in auth service")
}
func (s *service) EnsureRole(ctx context.Context, userID, roleID string) error {
	exists, err := s.userHasRole(ctx, userID, roleID)
	if err != nil || exists {
		return err
	}
	return s.changeRole(ctx, "add", userID, roleID)
}
func (s *service) RemoveRole(ctx context.Context, userID, roleID string) error {
	exists, err := s.userHasRole(ctx, userID, roleID)
	if err != nil || !exists {
		return err
	}
	return s.changeRole(ctx, "remove", userID, roleID)
}
func (s *service) changeRole(ctx context.Context, action, userID, roleID string) error {
	if strings.TrimSpace(s.apiKey) == "" {
		return fmt.Errorf("auth service API key is not configured")
	}
	data, err := json.Marshal(struct {
		RoleID string `json:"role_id"`
		UserID string `json:"user_id"`
	}{RoleID: roleID, UserID: userID})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.host+"/api/as/users/roles/"+action, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("build role request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: s.apiKey})
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("request auth role change: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("auth role change returned HTTP %d", resp.StatusCode)
	}
	if strings.HasPrefix(strings.ToLower(resp.Header.Get("Content-Type")), "text/html") {
		return fmt.Errorf("auth role change returned HTML instead of an API response")
	}
	_, err = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	return err
}
