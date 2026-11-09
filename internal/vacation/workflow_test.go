package vacation

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	repo "timetrack/internal/adapter/mysql/sqlc"
)

func TestDecisionTransitions(t *testing.T) {
	cases := []struct {
		name     string
		from, to repo.VacationsStatus
		comment  string
		status   int
	}{
		{"approve pending", repo.VacationsStatusPending, repo.VacationsStatusApproved, "", 0},
		{"reject with reason", repo.VacationsStatusPending, repo.VacationsStatusRejected, "Нужна замена", 0},
		{"reject without reason", repo.VacationsStatusPending, repo.VacationsStatusRejected, "", 0},
		{"reject whitespace", repo.VacationsStatusPending, repo.VacationsStatusRejected, " \n\t", 0},
		{"withdraw pending", repo.VacationsStatusPending, repo.VacationsStatusWithdrawn, "", 0},
		{"cannot withdraw approved", repo.VacationsStatusApproved, repo.VacationsStatusWithdrawn, "", 409},
		{"cannot approve twice", repo.VacationsStatusApproved, repo.VacationsStatusApproved, "", 409},
		{"cannot reject approved", repo.VacationsStatusApproved, repo.VacationsStatusRejected, "Причина", 409},
		{"return approved", repo.VacationsStatusApproved, repo.VacationsStatusPending, "", 0},
		{"return rejected", repo.VacationsStatusRejected, repo.VacationsStatusPending, "", 0},
		{"return withdrawn", repo.VacationsStatusWithdrawn, repo.VacationsStatusPending, "", 0},
		{"cannot return pending", repo.VacationsStatusPending, repo.VacationsStatusPending, "", 409},
		{"comment length", repo.VacationsStatusPending, repo.VacationsStatusRejected, strings.Repeat("я", 4001), 400},
		{"unicode comment boundary", repo.VacationsStatusPending, repo.VacationsStatusRejected, strings.Repeat("я", 4000), 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTransition(tc.from, tc.to, tc.comment)
			if tc.status == 0 {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var fe *fiber.Error
			if !errors.As(err, &fe) || fe.Code != tc.status {
				t.Fatalf("got %v, want HTTP %d", err, tc.status)
			}
		})
	}
}

type ownerFixture struct {
	Service
	edited bool
	actor  workflowActor
}

func (f *ownerFixture) GetVacationByID(context.Context, string) (*repo.GetVacationByIDRow, error) {
	return &repo.GetVacationByIDRow{ID: "application", UserID: "owner", Status: repo.VacationsStatusPending}, nil
}
func (f *ownerFixture) UpdateVacationStatus(ctx context.Context, _ string, status repo.VacationsStatus) error {
	f.actor = actorFrom(ctx)
	if status != repo.VacationsStatusWithdrawn {
		return errors.New("wrong status")
	}
	f.edited = true
	return nil
}
func TestOwnerWithdrawalCarriesAuthenticatedActor(t *testing.T) {
	fixture := &ownerFixture{}
	h := &Handler{service: fixture}
	app := fiber.New()
	app.Put("/:id/withdraw", func(c fiber.Ctx) error { c.Locals("user_id", "owner"); return h.Withdraw(c) })
	resp, err := app.Test(httptest.NewRequest("PUT", "/application/withdraw", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 || !fixture.edited || fixture.actor.ID != "owner" {
		t.Fatalf("status=%d, edited=%t, actor=%q", resp.StatusCode, fixture.edited, fixture.actor.ID)
	}
}

type historyFixture struct {
	ownerFixture
	events []repo.VacationHistory
}

func (f *historyFixture) GetHistory(context.Context, string) ([]repo.VacationHistory, error) {
	return f.events, nil
}
func TestHistoryWithInvalidLegacyDates(t *testing.T) {
	valid := time.Date(2026, 11, 9, 16, 0, 0, 0, time.UTC)
	dates := []time.Time{time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC), {}, time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC), valid}
	fixture := &historyFixture{}
	for i, date := range dates {
		fixture.events = append(fixture.events, repo.VacationHistory{ID: int64(i + 1), VacationID: "application", ActorID: "owner", EventType: "created", Detail: "Историческое событие", CreatedAt: date})
	}
	app := fiber.New()
	h := &Handler{service: fixture}
	app.Get("/:id/history", func(c fiber.Ctx) error { c.Locals("user_id", "owner"); return h.History(c) })
	resp, err := app.Test(httptest.NewRequest("GET", "/application/history", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var events []HistoryEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		t.Fatal(err)
	}
	if len(events) != len(dates) {
		t.Fatalf("lost events: got %d", len(events))
	}
	for i, event := range events {
		if event.Detail != "Историческое событие" {
			t.Fatal("event detail changed")
		}
		if i < 3 && event.CreatedAt != nil {
			t.Fatalf("invalid date %d was not null", i)
		}
	}
	if events[3].CreatedAt == nil || !events[3].CreatedAt.Equal(valid) {
		t.Fatal("valid timestamp changed")
	}
}
func TestEmptyHistoryIsArray(t *testing.T) {
	data, err := json.Marshal(historyResponse(nil))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "[]" {
		t.Fatalf("got %s", data)
	}
}

func TestRegisteredHistoryRoutePrecedesYearRoute(t *testing.T) {
	fixture := &historyFixture{}
	registered := fiber.New()
	SetupRoutes(registered, fixture, nil, nil, "time")
	// Keep the production route order and handlers, bypass only the remote auth
	// middleware: the fixture represents an already authenticated owner.
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error { c.Locals("user_id", "owner"); return c.Next() })
	for _, route := range registered.GetRoutes() {
		if route.Method == "GET" {
			app.Get(route.Path, route.Handlers[len(route.Handlers)-1])
		}
	}
	resp, err := app.Test(httptest.NewRequest("GET", "/vacation/application/history", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("history routed incorrectly: HTTP %d", resp.StatusCode)
	}
	var events []HistoryEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		t.Fatal(err)
	}
	if events == nil {
		t.Fatal("expected history array")
	}
}

func TestVacationYearRejectsInvalidParameters(t *testing.T) {
	app := fiber.New()
	h := &Handler{}
	app.Get("/:userId/:year", h.GetVacationsByYear)
	for _, year := range []string{"history", "0", "-1", "10000"} {
		resp, err := app.Test(httptest.NewRequest("GET", "/employee/"+year, nil))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 400 {
			t.Fatalf("year %q: got %d, want 400", year, resp.StatusCode)
		}
	}
}
