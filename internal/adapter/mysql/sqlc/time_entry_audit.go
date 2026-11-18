package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
	"timetrack/internal/timeaudit"
	"unicode/utf8"
)

// Keep legacy signatures while raw SQL is confined to the audited writers below.
type CreateUserTimeEntryParams = RawCreateUserTimeEntryParams
type UpdateUserTimeEntryParams = RawUpdateUserTimeEntryParams
type UpdateUserTimeEntriesParams = RawUpdateUserTimeEntriesParams
type DeleteUserTimeEntryParams = RawDeleteUserTimeEntryParams
type DeleteUserTimeEntriesParams = RawDeleteUserTimeEntriesParams

type TimeEntryKey struct {
	UserID string
	Date   time.Time
}
type monthKey struct {
	userID string
	year   int16
	month  int8
}

func entryDate(t time.Time) time.Time {
	// Match the existing MySQL driver location (loc=UTC) before truncating.
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
func timeEntryKeys(userID string, dates []time.Time) []TimeEntryKey {
	keys := make([]TimeEntryKey, 0, len(dates))
	for _, date := range dates {
		keys = append(keys, TimeEntryKey{UserID: userID, Date: date})
	}
	return keys
}

// LockTimeEntryMonths must run in the transaction that writes the entire batch.
func (q *Queries) LockTimeEntryMonths(ctx context.Context, keys []TimeEntryKey) error {
	if _, ok := q.db.(*sql.Tx); !ok {
		return errors.New("timesheet month locks require a transaction")
	}
	months := map[monthKey]bool{}
	for _, key := range keys {
		key.Date = entryDate(key.Date)
		if key.UserID == "" || key.Date.IsZero() || key.Date.Year() < 1 || key.Date.Year() > 9999 {
			return errors.New("invalid time entry owner or date")
		}
		months[monthKey{key.UserID, int16(key.Date.Year()), int8(key.Date.Month())}] = true
	}
	ordered := make([]monthKey, 0, len(months))
	for key := range months {
		ordered = append(ordered, key)
	}
	sort.Slice(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if a.userID != b.userID {
			return a.userID < b.userID
		}
		if a.year != b.year {
			return a.year < b.year
		}
		return a.month < b.month
	})
	for _, key := range ordered {
		if err := q.EnsureTimesheetMonth(ctx, EnsureTimesheetMonthParams{UserID: key.userID, Year: key.year, Month: key.month}); err != nil {
			return err
		}
		if _, err := q.LockTimesheetMonth(ctx, LockTimesheetMonthParams{UserID: key.userID, Year: key.year, Month: key.month}); err != nil {
			return err
		}
	}
	return nil
}
func (q *Queries) LockTimeEntryRange(ctx context.Context, userID string, start, end time.Time) error {
	start, end = start.UTC(), end.UTC()
	keys := []TimeEntryKey{}
	for month := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC); !month.After(entryDate(end)); month = month.AddDate(0, 1, 0) {
		keys = append(keys, TimeEntryKey{UserID: userID, Date: month})
	}
	return q.LockTimeEntryMonths(ctx, keys)
}
func (q *Queries) mutateTimeEntries(ctx context.Context, keys []TimeEntryKey, write func(*Queries) error) error {
	if err := timeaudit.Validate(ctx); err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}
	if db, ok := q.db.(*sql.DB); ok {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if err := q.WithTx(tx).mutateTimeEntries(ctx, keys, write); err != nil {
			return err
		}
		return tx.Commit()
	}
	if err := q.LockTimeEntryMonths(ctx, keys); err != nil {
		return err
	}
	actor := timeaudit.FromContext(ctx)
	switch actor.Type {
	case "employee", "manager", "admin", "system":
	default:
		return errors.New("invalid time audit actor type")
	}
	if utf8.RuneCountInString(actor.Comment) > 4000 {
		return errors.New("time audit comment exceeds 4000 characters")
	}
	unique := map[TimeEntryKey]bool{}
	for _, key := range keys {
		key.Date = entryDate(key.Date)
		unique[key] = true
	}
	keys = make([]TimeEntryKey, 0, len(unique))
	for key := range unique {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].UserID != keys[j].UserID {
			return keys[i].UserID < keys[j].UserID
		}
		return keys[i].Date.Before(keys[j].Date)
	})
	before := make(map[TimeEntryKey]*UserTimeEntry, len(keys))
	for _, key := range keys {
		value, err := q.timeEntrySnapshot(ctx, key)
		if err != nil {
			return err
		}
		before[key] = value
		if actor.Type == "manager" {
			if !timeaudit.ManagerOwns(ctx, key.UserID) {
				return timeaudit.ErrProtectedEntry
			}
			if err := q.managerWritable(ctx, key, value); err != nil {
				return err
			}
		}
	}
	if err := write(q); err != nil {
		return err
	}
	for _, key := range keys {
		after, err := q.timeEntrySnapshot(ctx, key)
		if err != nil {
			return err
		}
		if actor.Type == "manager" && after != nil {
			if err := q.managerWorkType(ctx, after.DayTypeID); err != nil {
				return err
			}
		}
		old := before[key]
		if sameTimeEntry(old, after) {
			continue
		}
		year, month := int16(key.Date.Year()), int8(key.Date.Month())
		if err := q.ReviseTimesheetMonth(ctx, ReviseTimesheetMonthParams{UserID: key.UserID, Year: year, Month: month}); err != nil {
			return err
		}
		state, err := q.LockTimesheetMonth(ctx, LockTimesheetMonthParams{UserID: key.UserID, Year: year, Month: month})
		if err != nil {
			return err
		}
		operation := TimeEntryHistoryOperationUpdate
		entryID := ""
		if old != nil {
			entryID = old.ID
		}
		if after != nil {
			entryID = after.ID
		}
		if old == nil {
			operation = TimeEntryHistoryOperationCreate
		} else if after == nil {
			operation = TimeEntryHistoryOperationDelete
		}
		oldJSON, err := snapshotJSON(old)
		if err != nil {
			return err
		}
		newJSON, err := snapshotJSON(after)
		if err != nil {
			return err
		}
		if err := q.AppendTimeEntryHistory(ctx, AppendTimeEntryHistoryParams{
			UserID: key.UserID, EntryDate: key.Date, EntryID: sql.NullString{String: entryID, Valid: entryID != ""},
			ActorID: sql.NullString{String: actor.ID, Valid: actor.ID != ""}, ActorType: TimeEntryHistoryActorType(actor.Type), Source: actor.Source,
			Operation: operation, BeforeValues: oldJSON, AfterValues: newJSON, Comment: actor.Comment, MonthRevision: state.Revision,
		}); err != nil {
			return err
		}
	}
	return nil
}
func (q *Queries) timeEntrySnapshot(ctx context.Context, key TimeEntryKey) (*UserTimeEntry, error) {
	row, err := q.LockTimeEntryByDate(ctx, LockTimeEntryByDateParams{UserID: key.UserID, EntryDate: key.Date})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}
func sameTimeEntry(a, b *UserTimeEntry) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.ID == b.ID && a.DayTypeID == b.DayTypeID && a.HoursWorked == b.HoursWorked && a.WorkLocation == b.WorkLocation
}
func snapshotJSON(entry *UserTimeEntry) (json.RawMessage, error) {
	if entry == nil {
		return nil, nil
	}
	var location *string
	if entry.WorkLocation.Valid {
		value := entry.WorkLocation.String
		location = &value
	}
	data, err := json.Marshal(struct {
		EntryID      string  `json:"entryId"`
		DayTypeID    string  `json:"dayTypeId"`
		HoursWorked  string  `json:"hoursWorked"`
		WorkLocation *string `json:"workLocation"`
	}{entry.ID, entry.DayTypeID, entry.HoursWorked, location})
	if err != nil {
		return nil, fmt.Errorf("encode time entry audit: %w", err)
	}
	return data, nil
}
func (q *Queries) CreateUserTimeEntry(ctx context.Context, arg CreateUserTimeEntryParams) error {
	arg.EntryDate = entryDate(arg.EntryDate)
	return q.mutateTimeEntries(ctx, []TimeEntryKey{{arg.UserID, arg.EntryDate}}, func(tx *Queries) error { return tx.RawCreateUserTimeEntry(ctx, arg) })
}
func (q *Queries) UpdateUserTimeEntry(ctx context.Context, arg UpdateUserTimeEntryParams) error {
	arg.EntryDate = entryDate(arg.EntryDate)
	return q.mutateTimeEntries(ctx, []TimeEntryKey{{arg.UserID, arg.EntryDate}}, func(tx *Queries) error { return tx.RawUpdateUserTimeEntry(ctx, arg) })
}
func (q *Queries) UpdateUserTimeEntries(ctx context.Context, arg UpdateUserTimeEntriesParams) error {
	dates := make([]time.Time, len(arg.EntryDate))
	for i, date := range arg.EntryDate {
		dates[i] = entryDate(date)
	}
	arg.EntryDate = dates
	return q.mutateTimeEntries(ctx, timeEntryKeys(arg.UserID, dates), func(tx *Queries) error { return tx.RawUpdateUserTimeEntries(ctx, arg) })
}
func (q *Queries) DeleteUserTimeEntry(ctx context.Context, arg DeleteUserTimeEntryParams) error {
	arg.EntryDate = entryDate(arg.EntryDate)
	return q.mutateTimeEntries(ctx, []TimeEntryKey{{arg.UserID, arg.EntryDate}}, func(tx *Queries) error { return tx.RawDeleteUserTimeEntry(ctx, arg) })
}
func (q *Queries) DeleteUserTimeEntries(ctx context.Context, arg DeleteUserTimeEntriesParams) error {
	dates := make([]time.Time, len(arg.EntryDate))
	for i, date := range arg.EntryDate {
		dates[i] = entryDate(date)
	}
	arg.EntryDate = dates
	return q.mutateTimeEntries(ctx, timeEntryKeys(arg.UserID, dates), func(tx *Queries) error { return tx.RawDeleteUserTimeEntries(ctx, arg) })
}

func (q *Queries) managerWorkType(ctx context.Context, id string) error {
	dt, err := q.GetDayTypeByID(ctx, id)
	if err != nil {
		return err
	}
	if dt.SystemName != "work" {
		return timeaudit.ErrProtectedEntry
	}
	return nil
}
func (q *Queries) managerWritable(ctx context.Context, key TimeEntryKey, entry *UserTimeEntry) error {
	if entry == nil {
		return nil
	}
	if err := q.managerWorkType(ctx, entry.DayTypeID); err != nil {
		return err
	}
	protected, err := q.ManagerProtectedTimeEntry(ctx, ManagerProtectedTimeEntryParams{UserID: key.UserID, EntryDate: key.Date, EntryID: sql.NullString{String: entry.ID, Valid: true}})
	if err != nil {
		return err
	}
	if protected > 0 {
		return timeaudit.ErrProtectedEntry
	}
	return nil
}
