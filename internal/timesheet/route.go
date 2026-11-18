package timesheet

import (
	"database/sql"
	"github.com/gofiber/fiber/v3"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/authservice"
	"timetrack/internal/timeaccess"
	usertimeentry "timetrack/internal/user_time_entry"
)

func SetupRoutes(router fiber.Router, db *sql.DB, policy *timeaccess.Policy, users authservice.Service, entries usertimeentry.Service) {
	h := &Handler{service: NewService(repo.New(db)), db: db, policy: policy, users: users, entries: entries}
	routes := router.Group("/timesheets", func(c fiber.Ctx) error { c.Set("Cache-Control", "no-store"); return c.Next() })
	routes.Get("/:year/:month", h.List)
	routes.Get("/:userId/:year/:month", h.Month)
	routes.Get("/:userId/:year/:month/history", h.History)
	routes.Post("/:userId/:year/:month/approve", h.decision(true))
	routes.Post("/:userId/:year/:month/unapprove", h.decision(false))
	routes.Put("/:userId/:year/:month/entries", h.Edit)
}
