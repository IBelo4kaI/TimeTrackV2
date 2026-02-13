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
    SUM(hours_worked) as total_hours
FROM user_time_entries
WHERE user_id = sqlc.arg(user_id) AND YEAR(entry_date) = YEAR(sqlc.arg(year)) AND MONTH(entry_date) = MONTH(sqlc.arg(month))
GROUP BY user_id, YEAR(entry_date), MONTH(entry_date);

-- name: GetTotalHoursByYear :one
SELECT
    SUM(hours_worked) as total_hours
FROM user_time_entries
WHERE user_id = sqlc.arg(user_id) AND YEAR(entry_date) = YEAR(sqlc.arg(year))
GROUP BY user_id, YEAR(entry_date);

-- name: GetWorkDaysByMonth :one
SELECT
    COUNT(DISTINCT entry_date) as total_days
FROM user_time_entries
WHERE user_id = sqlc.arg(user_id)
    AND YEAR(entry_date) = YEAR(sqlc.arg(year))
    AND MONTH(entry_date) = MONTH(sqlc.arg(month))
    AND hours_worked > 0;

-- name: GetTotalDaysByMonthWithSystemName :one
SELECT
    COUNT(ute.entry_date) as total_days
FROM user_time_entries ute
JOIN day_types dt ON ute.day_type_id = dt.id
WHERE ute.user_id = sqlc.arg(user_id)
    AND YEAR(ute.entry_date) = YEAR(sqlc.arg(year))
    AND MONTH(ute.entry_date) = MONTH(sqlc.arg(month))
    AND dt.system_name = sqlc.arg(system_name)
GROUP BY ute.user_id, YEAR(ute.entry_date), MONTH(ute.entry_date);

-- name: GetTotalDaysByYearWithSystemName :one
SELECT
    COUNT(ute.entry_date) as total_days
FROM user_time_entries ute
JOIN day_types dt ON ute.day_type_id = dt.id
WHERE ute.user_id = sqlc.arg(user_id)
    AND YEAR(ute.entry_date) = YEAR(sqlc.arg(year))
    AND dt.system_name = sqlc.arg(system_name)
GROUP BY ute.user_id, YEAR(ute.entry_date);

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
