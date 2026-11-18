package timesheet

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v3"
)

var historyScanColumn = regexp.MustCompile(`^sql: Scan error on column index ([0-9]+), name "(id|user_id|entry_date|entry_id|actor_id|actor_type|source|operation|before_values|after_values|comment|month_revision|created_at)":`)

func (h *Handler) historyError(c fiber.Ctx, stage string, err error) error {
	var api *fiber.Error
	if errors.As(err, &api) && api.Code < 500 {
		return apiError(c, err)
	}
	logger := h.logger
	if logger == nil {
		logger = slog.Default()
	}
	attrs := []any{"stage", stage, "error_type", fmt.Sprintf("%T", err)}
	var dbErr *mysql.MySQLError
	switch {
	case errors.As(err, &dbErr):
		attrs = append(attrs, "cause", "mysql", "mysql_number", dbErr.Number, "sqlstate", string(dbErr.SQLState[:]))
	case errors.Is(err, context.DeadlineExceeded):
		attrs = append(attrs, "cause", "deadline_exceeded")
	case errors.Is(err, context.Canceled):
		attrs = append(attrs, "cause", "context_canceled")
	case errors.Is(err, driver.ErrBadConn), errors.Is(err, mysql.ErrInvalidConn):
		attrs = append(attrs, "cause", "invalid_connection")
	default:
		// Database error text can contain values; log only known scan metadata.
		if column := historyScanColumn.FindStringSubmatch(err.Error()); column != nil {
			cause := "sql_scan"
			if strings.Contains(err.Error(), "unsupported Scan") && strings.Contains(err.Error(), "<nil>") {
				cause = "sql_scan_null"
			}
			attrs = append(attrs, "cause", cause, "column_index", column[1], "column", column[2])
		} else if api != nil {
			attrs = append(attrs, "cause", "http_dependency", "http_status", api.Code)
		} else {
			attrs = append(attrs, "cause", "internal")
		}
	}
	logger.Error("timesheet history failed", attrs...)
	return apiError(c, err)
}
