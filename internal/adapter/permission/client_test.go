package permission

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestValidate(t *testing.T) {
	for _, tt := range []struct {
		name        string
		target      string
		permissions string
		user        string
		status      int
		allowed     bool
		wantError   bool
	}{
		{name: "own record", target: "caller", permissions: `[{"code":"time:vacation:read"}]`, allowed: true},
		{name: "no target", permissions: `[{"code":"time:vacation:read"}]`, allowed: true},
		{name: "other record denied", target: "other", permissions: `[{"code":"time:vacation:read"}]`},
		{name: "other record allowed", target: "other", permissions: `[{"code":"time:vacation.all:read"}]`, allowed: true},
		{name: "all scope", target: "all-scope", permissions: `[{"code":"time:vacation.all:read"}]`, allowed: true},
		{name: "service wildcard", target: "other", permissions: `[{"code":"time:all:all"}]`, allowed: true},
		{name: "global wildcard", target: "other", permissions: `[{"code":"all:all:all"}]`, allowed: true},
		{name: "action wildcard", permissions: `[{"code":"time:vacation:all"}]`, allowed: true},
		{name: "wrong service", permissions: `[{"code":"other:vacation:read"}]`},
		{name: "wrong action", permissions: `[{"code":"time:vacation:edit"}]`},
		{name: "empty permissions", permissions: `[]`},
		{name: "unavailable", status: http.StatusServiceUnavailable, wantError: true},
		{name: "expired session", status: http.StatusUnauthorized, wantError: true},
		{name: "invalid JSON", permissions: `{`, wantError: true},
		{name: "missing identity", user: `{}`, wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient("http://auth:8382", "service-id")
			if err != nil {
				t.Fatal(err)
			}
			client.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				cookie, err := r.Cookie("session")
				if err != nil || cookie.Value != "user-session" {
					t.Fatal("request must use the caller's session")
				}
				if r.Method != http.MethodGet || r.URL.Host != "auth:8382" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
				}
				status := http.StatusOK
				var body string
				switch r.URL.Path {
				case "/api/as/users/me":
					body = tt.user
					if body == "" {
						body = `{"id":"caller"}`
					}
				case "/api/as/users/me/permissions/service-id":
					body = tt.permissions
					if tt.status != 0 {
						status = tt.status
					}
				default:
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			resp, err := client.Validate(context.Background(), &PermissionRequest{
				SessionToken: "user-session", Service: "time", Entity: "vacation", Action: "read", UserId: &tt.target,
			})
			if (err != nil) != tt.wantError {
				t.Fatalf("error = %v, wantError = %v", err, tt.wantError)
			}
			if tt.wantError {
				return
			}
			if resp.IsAccess != tt.allowed || resp.UserId != "caller" {
				t.Fatalf("unexpected result: %+v", resp)
			}
		})
	}
}
