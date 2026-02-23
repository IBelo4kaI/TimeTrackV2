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

-- name: GetTotalHoursByMonth :one
SELECT
    COALESCE(SUM(hours_worked), 0) as total_hours
FROM user_time_entries
WHERE user_id = sqlc.arg(user_id) AND YEAR(entry_date) = YEAR(sqlc.arg(year)) AND MONTH(entry_date) = MONTH(sqlc.arg(month));

-- name: GetTotalHoursByYear :one
SELECT
    COALESCE(SUM(hours_worked), 0) as total_hours
FROM user_time_entries
WHERE user_id = sqlc.arg(user_id) AND YEAR(entry_date) = YEAR(sqlc.arg(year));

-- name: GetWorkDaysByMonth :one
SELECT
    COALESCE(COUNT(DISTINCT entry_date), 0) as total_days
FROM user_time_entries
WHERE user_id = sqlc.arg(user_id)
    AND YEAR(entry_date) = YEAR(sqlc.arg(year))
    AND MONTH(entry_date) = MONTH(sqlc.arg(month))
    AND hours_worked > 0;

-- name: GetTotalDaysByMonthWithSystemName :one
SELECT
    COALESCE(COUNT(ute.entry_date), 0) as total_days
FROM user_time_entries ute
JOIN day_types dt ON ute.day_type_id = dt.id
WHERE ute.user_id = sqlc.arg(user_id)
    AND YEAR(ute.entry_date) = YEAR(sqlc.arg(year))
    AND MONTH(ute.entry_date) = MONTH(sqlc.arg(month))
    AND dt.system_name = sqlc.arg(system_name);

-- name: GetTotalDaysByYearWithSystemName :one
SELECT
    COALESCE(COUNT(ute.entry_date), 0) as total_days
FROM user_time_entries ute
JOIN day_types dt ON ute.day_type_id = dt.id
WHERE ute.user_id = sqlc.arg(user_id)
    AND YEAR(ute.entry_date) = YEAR(sqlc.arg(year))
    AND dt.system_name = sqlc.arg(system_name);

-- name: GetVacationDaysByYear :one
SELECT
    COALESCE(COUNT(ute.entry_date), 0) as used_vacation_days
FROM user_time_entries ute
JOIN day_types dt ON ute.day_type_id = dt.id
WHERE ute.user_id = sqlc.arg(user_id)
    AND YEAR(ute.entry_date) = YEAR(sqlc.arg(year))
    AND dt.system_name = 'vacation';

-- name: GetVacationDaysByMonth :one
SELECT
    COALESCE(COUNT(ute.entry_date), 0) as used_vacation_days
FROM user_time_entries ute
JOIN day_types dt ON ute.day_type_id = dt.id
WHERE ute.user_id = sqlc.arg(user_id)
    AND YEAR(ute.entry_date) = YEAR(sqlc.arg(year))
    AND MONTH(ute.entry_date) = MONTH(sqlc.arg(month))
    AND dt.system_name = 'vacation';

-- name: CreateUserTimeEntry :exec
INSERT INTO user_time_entries (user_id, entry_date, day_type_id, hours_worked)
VALUES (?, ?, ?, ?);

-- name: UpdateUserTimeEntry :exec
UPDATE user_time_entries
SET
    day_type_id = ?,
    hours_worked = ?
WHERE entry_date = ? AND user_id = ?;

-- name: UpdateUserTimeEntries :exec
UPDATE user_time_entries
SET
    day_type_id = ?,
    hours_worked = ?
WHERE entry_date IN (sqlc.slice('entry_date')) AND user_id = ?;

-- name: DeleteUserTimeEntry :exec
DELETE FROM user_time_entries WHERE entry_date = ? AND user_id = ?;

-- name: DeleteUserTimeEntries :exec
DELETE FROM user_time_entries WHERE entry_date IN (sqlc.slice('entry_date')) AND user_id = ?;
