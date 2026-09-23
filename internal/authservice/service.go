package authservice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Service — HTTP-клиент к сервису авторизации (отдельному от timetrack,
// см. AUTH_SERVICE_HOST/AUTH_SERVICE_API_KEY в env), не gRPC — тот уже
// используется только для проверки прав (см. internal/adapter/grpc).
type Service interface {
	GetAllUsers(ctx context.Context) ([]UserResponse, error)
}

type service struct {
	host   string
	apiKey string
	client *http.Client
}

// NewService — apiKey отправляется как значение cookie "session" (не
// заголовок), так авторизуется сервис авторизации.
func NewService(host, apiKey string) Service {
	return &service{
		host:   strings.TrimRight(host, "/"),
		apiKey: apiKey,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *service) GetAllUsers(ctx context.Context) ([]UserResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.host+"/api/as/users/all", nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.AddCookie(&http.Cookie{Name: "session", Value: s.apiKey})

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request users: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth service returned status %d", resp.StatusCode)
	}

	var users []UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return users, nil
}
