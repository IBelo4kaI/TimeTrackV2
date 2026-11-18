-- name: GetTimesheetMonth :one
SELECT * FROM timesheet_months WHERE user_id = ? AND year = ? AND month = ?;

-- name: EnsureTimesheetMonth :exec
INSERT INTO timesheet_months (user_id, year, month) VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE user_id = VALUES(user_id);

-- name: LockTimesheetMonth :one
SELECT * FROM timesheet_months WHERE user_id = ? AND year = ? AND month = ? FOR UPDATE;

-- name: ReviseTimesheetMonth :exec
UPDATE timesheet_months SET revision = revision + 1,
 status = CASE WHEN status = 'approved' THEN 'needs_review' ELSE status END
WHERE user_id = ? AND year = ? AND month = ?;

-- name: AppendTimeEntryHistory :exec
INSERT INTO time_entry_history
(user_id, entry_date, entry_id, actor_id, actor_type, source, operation, before_values, after_values, comment, month_revision)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListTimeEntryHistory :many
SELECT * FROM time_entry_history WHERE user_id = ? AND entry_date >= sqlc.arg(from_date) AND entry_date < sqlc.arg(to_date) ORDER BY id;

-- name: LockTimeEntryByDate :one
SELECT * FROM user_time_entries WHERE user_id = ? AND entry_date = ? FOR UPDATE;

-- name: LockTimeEntriesForMonth :many
SELECT * FROM user_time_entries WHERE user_id = ? AND entry_date >= sqlc.arg(from_date) AND entry_date < sqlc.arg(to_date) ORDER BY entry_date FOR UPDATE;

-- name: ApproveTimesheetMonth :exec
UPDATE timesheet_months SET status = 'approved', approved_revision = revision,
 approved_by = ?, approved_at = CURRENT_TIMESTAMP
WHERE user_id = ? AND year = ? AND month = ?;

-- name: UnapproveTimesheetMonth :exec
UPDATE timesheet_months SET status = 'pending', approved_revision = NULL,
 approved_by = NULL, approved_at = NULL
WHERE user_id = ? AND year = ? AND month = ?;

-- name: ManagerProtectedTimeEntry :one
SELECT COUNT(*) FROM time_entry_history
WHERE user_id = ? AND entry_date = ? AND entry_id = ?
 AND (actor_type = 'system' OR source IN ('vacation', 'sick_leave'));
