-- ============================================
-- user_time_entries queries
-- ============================================

-- name: GetUserTimeEntriesForMonth :many
SELECT *
FROM user_time_entries
WHERE user_id = sqlc.arg(user_id) AND YEAR(entry_date) = YEAR(sqlc.arg(year)) AND MONTH(entry_date) = MONTH(sqlc.arg(month))
ORDER BY entry_date;

-- name: GetUserTimeEntryById :one
SELECT *
FROM user_time_entries
WHERE id = ?;

-- name: GetUserTimeEntryByIds :many
SELECT *
FROM user_time_entries
WHERE id IN (sqlc.slice('ids'));

-- name: GetTotalHoursForUserTimeEntriesByMonth :many
SELECT
    user_id,
    YEAR(entry_date) as year,
    MONTH(entry_date) as month,
    SUM(hours_worked) as total_hours
FROM user_time_entries
WHERE user_id = sqlc.arg(user_id) AND YEAR(entry_date) = sqlc.arg(year) AND MONTH(entry_date) = sqlc.arg(month)
GROUP BY user_id, YEAR(entry_date), MONTH(entry_date);

-- name: GetTotalHoursForUserTimeEntriesByYear :many
SELECT
    user_id,
    YEAR(entry_date) as year,
    MONTH(entry_date) as month,
    SUM(hours_worked) as total_hours
FROM user_time_entries
WHERE user_id = sqlc.arg(user_id) AND YEAR(entry_date) = sqlc.arg(year)
GROUP BY user_id, YEAR(entry_date), MONTH(entry_date)
ORDER BY MONTH(entry_date);

-- name: CreateUserTimeEntry :exec
INSERT INTO user_time_entries (user_id, entry_date, day_type_id, hours_worked)
VALUES (?, ?, ?, ?);

-- name: UpdateUserTimeEntry :exec
UPDATE user_time_entries
SET
    day_type_id = ?,
    hours_worked = ?
WHERE id = ?;

-- name: UpdateUserTimeEntries :exec
UPDATE user_time_entries
SET
    day_type_id = ?,
    hours_worked = ?
WHERE id IN (sqlc.slice('ids'));

-- name: DeleteUserTimeEntry :exec
DELETE FROM user_time_entries WHERE id = ?;

-- name: DeleteUserTimeEntries :exec
DELETE FROM user_time_entries WHERE id IN (sqlc.slice('ids'));
