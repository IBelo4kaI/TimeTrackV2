package timesheet

import (
	"database/sql"
	"errors"
	"github.com/gofiber/fiber/v3"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/authservice"
	"timetrack/internal/response"
	"timetrack/internal/timeaccess"
	"timetrack/internal/timeaudit"
	usertimeentry "timetrack/internal/user_time_entry"
	"unicode/utf8"
)

type Handler struct {
	logger  *slog.Logger
	service *Service
	db      *sql.DB
	policy  *timeaccess.Policy
	users   authservice.Service
	entries usertimeentry.Service
}

type monthResponse struct {
	RequiresApproval bool                       `json:"requiresApproval"`
	UserID           string                     `json:"userId"`
	Year             int16                      `json:"year"`
	Month            int8                       `json:"month"`
	Status           repo.TimesheetMonthsStatus `json:"status"`
	Revision         int64                      `json:"revision"`
	ApprovedRevision *int64                     `json:"approvedRevision"`
	ApprovedBy       *string                    `json:"approvedBy"`
	ApprovedAt       *time.Time                 `json:"approvedAt"`
}

func monthDTO(state repo.TimesheetMonth, requiresApproval bool) monthResponse {
	result := monthResponse{RequiresApproval: requiresApproval, UserID: state.UserID, Year: state.Year, Month: state.Month, Status: state.Status, Revision: state.Revision}
	if state.ApprovedRevision.Valid {
		result.ApprovedRevision = &state.ApprovedRevision.Int64
	}
	if state.ApprovedBy.Valid {
		result.ApprovedBy = &state.ApprovedBy.String
	}
	if state.ApprovedAt.Valid {
		result.ApprovedAt = &state.ApprovedAt.Time
	}
	return result
}
func period(c fiber.Ctx) (int, int, error) {
	year, e1 := strconv.Atoi(c.Params("year"))
	month, e2 := strconv.Atoi(c.Params("month"))
	if e1 != nil || e2 != nil || year < 1 || year > 9999 || month < 1 || month > 12 {
		return 0, 0, fiber.ErrBadRequest
	}
	return year, month, nil
}
func apiError(c fiber.Ctx, err error) error {
	var api *fiber.Error
	if errors.As(err, &api) {
		return response.Error(c, api.Code, api)
	}
	if errors.Is(err, timeaudit.ErrProtectedEntry) {
		return response.Error(c, 403, fiber.NewError(403, "Руководитель может изменять только рабочие записи"))
	}
	if errors.Is(err, timeaudit.ErrUnauthenticated) {
		return fiber.ErrUnauthorized
	}
	return response.ServerError(c)
}
func (h *Handler) Month(c fiber.Ctx) error {
	year, month, err := period(c)
	if err != nil {
		return err
	}
	if _, err = h.policy.Authorize(c, c.Params("userId"), "read", ""); err != nil {
		return apiError(c, err)
	}
	state, err := h.service.GetMonth(c.RequestCtx(), c.Params("userId"), year, month)
	if err != nil {
		return apiError(c, err)
	}
	required, err := h.service.RequiresApproval(c.RequestCtx(), state.UserID)
	if err != nil {
		return apiError(c, err)
	}
	return response.Success(c, monthDTO(state, required))
}
func (h *Handler) History(c fiber.Ctx) error {
	year, month, err := period(c)
	if err != nil {
		return err
	}
	if _, err = h.policy.Authorize(c, c.Params("userId"), "read", ""); err != nil {
		return h.historyError(c, "authorize", err)
	}
	rows, err := h.service.History(c.RequestCtx(), c.Params("userId"), year, month)
	if err != nil {
		return h.historyError(c, "read_history", err)
	}
	if err := response.Success(c, rows); err != nil {
		return h.historyError(c, "encode_response", err)
	}
	return nil
}
func (h *Handler) decision(approve bool) fiber.Handler {
	return func(c fiber.Ctx) error {
		year, month, err := period(c)
		if err != nil {
			return err
		}
		ctx, err := h.policy.Authorize(c, c.Params("userId"), "approve", "")
		if err != nil {
			return apiError(c, err)
		}
		var body struct {
			ExpectedRevision *int64 `json:"expectedRevision"`
		}
		if err := c.Bind().Body(&body); err != nil || body.ExpectedRevision == nil || *body.ExpectedRevision < 0 {
			return fiber.ErrBadRequest
		}
		required, err := h.service.RequiresApproval(c.RequestCtx(), c.Params("userId"))
		if err != nil {
			return apiError(c, err)
		}
		state, err := h.service.Decide(ctx, h.db, c.Params("userId"), year, month, *body.ExpectedRevision, approve)
		if err != nil {
			var api *fiber.Error
			if errors.As(err, &api) && api.Code == 409 {
				return c.Status(409).JSON(fiber.Map{"error": api.Message, "current": monthDTO(state, required)})
			}
			return apiError(c, err)
		}
		return response.Success(c, monthDTO(state, required))
	}
}

type listRow struct {
	monthResponse
	Name                  string                                  `json:"name"`
	Surname               string                                  `json:"surname"`
	Patronymic            *string                                 `json:"patronymic"`
	Statistics            *usertimeentry.ReportStatisticsResponse `json:"statistics"`
	StatisticsUnavailable bool                                    `json:"statisticsUnavailable"`
	CanEdit               bool                                    `json:"canEdit"`
	CanApprove            bool                                    `json:"canApprove"`
}

func (h *Handler) List(c fiber.Ctx) error {
	year, month, err := period(c)
	if err != nil {
		return err
	}
	actor, err := h.policy.Identity(c)
	if err != nil {
		return apiError(c, err)
	}
	allRead := h.policy.HasAll(c, actor, "calendar", "read")
	var members map[string]bool
	if !allRead {
		members, err = h.policy.Members(c, actor)
		if err != nil {
			return apiError(c, err)
		}
	}
	users, err := h.users.GetAllUsers(c.RequestCtx())
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	byID := map[string]authservice.UserResponse{}
	for _, user := range users {
		if user.ID != "" && (allRead || members[user.ID]) {
			byID[user.ID] = user
		}
	}
	ids := []string{}
	if allRead {
		for id := range byID {
			ids = append(ids, id)
		}
	} else {
		for id := range members {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	allEdit := h.policy.HasAll(c, actor, "calendar", "edit")
	allApprove := h.policy.HasAll(c, actor, "timesheet", "approve")
	// Scope is resolved once per request, including for administrators with a manager role.
	if allRead && (!allEdit || !allApprove) {
		members, _ = h.policy.Members(c, actor)
	}
	exempt, err := h.service.ApprovalExemptUsers(c.RequestCtx())
	if err != nil {
		return apiError(c, err)
	}
	rows := make([]listRow, 0, len(ids))
	for _, id := range ids {
		state, e := h.service.GetMonth(c.RequestCtx(), id, year, month)
		if e != nil {
			return apiError(c, e)
		}
		user := byID[id]
		row := listRow{monthResponse: monthDTO(state, !exempt[id]), Name: user.Name, Surname: user.Surname, Patronymic: user.Patronymic, CanEdit: allEdit || members[id], CanApprove: allApprove || members[id]}
		gender, e := strconv.Atoi(user.Gender.ID)
		if e != nil || gender < 1 {
			row.StatisticsUnavailable = true
		} else {
			row.Statistics, e = h.entries.GetReportStatistics(c.RequestCtx(), id, month, year, gender)
			if e != nil {
				return apiError(c, e)
			}
		}
		rows = append(rows, row)
	}
	return response.Success(c, rows)
}
func (h *Handler) Edit(c fiber.Ctx) error {
	year, month, err := period(c)
	if err != nil {
		return err
	}
	var body struct {
		Operation string `json:"operation"`
		Comment   string `json:"comment"`
		Entities  []struct {
			EntryDate    time.Time `json:"entryDate"`
			DayTypeID    string    `json:"dayTypeId"`
			HoursWorked  string    `json:"hoursWorked"`
			WorkLocation *string   `json:"workLocation"`
		} `json:"entities"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return fiber.ErrBadRequest
	}
	if body.Operation != "create" && body.Operation != "update" && body.Operation != "delete" {
		return fiber.ErrBadRequest
	}
	if len(body.Entities) == 0 || utf8.RuneCountInString(body.Comment) > 4000 {
		return fiber.ErrBadRequest
	}
	// This endpoint edits other employees; self-service keeps its original routes.
	actor, err := h.policy.Identity(c)
	if err != nil {
		return apiError(c, err)
	}
	target := c.Params("userId")
	action := "edit"
	if body.Operation == "create" {
		action = "create"
	} else if body.Operation == "delete" {
		action = "delete"
	}
	ctx, err := h.policy.Authorize(c, target, action, strings.TrimSpace(body.Comment))
	if err != nil {
		return apiError(c, err)
	}
	if actor == target && !h.policy.HasAll(c, actor, "calendar", action) {
		return fiber.ErrForbidden
	}
	creates := []repo.CreateUserTimeEntryParams{}
	updates := []repo.UpdateUserTimeEntryParams{}
	dates := []time.Time{}
	for _, entity := range body.Entities {
		if entity.EntryDate.UTC().Year() != year || int(entity.EntryDate.UTC().Month()) != month || entity.EntryDate.IsZero() {
			return fiber.ErrBadRequest
		}
		if entity.WorkLocation != nil && *entity.WorkLocation != "office" && *entity.WorkLocation != "remote" {
			return fiber.ErrBadRequest
		}
		loc := sql.NullString{}
		if entity.WorkLocation != nil {
			loc = sql.NullString{String: *entity.WorkLocation, Valid: true}
		}
		if body.Operation != "delete" {
			if strings.TrimSpace(entity.DayTypeID) == "" {
				return fiber.ErrBadRequest
			}
		}
		creates = append(creates, repo.CreateUserTimeEntryParams{UserID: target, EntryDate: entity.EntryDate, DayTypeID: entity.DayTypeID, HoursWorked: entity.HoursWorked, WorkLocation: loc})
		updates = append(updates, repo.UpdateUserTimeEntryParams{UserID: target, EntryDate: entity.EntryDate, DayTypeID: entity.DayTypeID, HoursWorked: entity.HoursWorked, WorkLocation: loc})
		dates = append(dates, entity.EntryDate)
	}
	switch body.Operation {
	case "create":
		err = h.entries.CreateUserTimeEntry(ctx, creates)
	case "update":
		err = h.entries.UpdateUserTimeEntries(ctx, updates)
	case "delete":
		err = h.entries.DeleteUserTimeEntries(ctx, repo.DeleteUserTimeEntriesParams{UserID: target, EntryDate: dates})
	}
	if err != nil {
		return apiError(c, err)
	}
	return response.Updated(c)
}
