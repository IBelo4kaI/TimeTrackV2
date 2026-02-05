-- ============================================
-- calendar_events queries
-- ============================================

-- name: GetCalendarEventsForMonth :many
SELECT
    id,
    event_date,
    day_type_id,
    COALESCE(description, '') as description,
    created_at,
    updated_at
FROM calendar_events
WHERE YEAR(event_date) = YEAR(sqlc.arg(year)) AND MONTH(event_date) = MONTH(sqlc.arg(month))
ORDER BY event_date;

-- name: GetCalendarEventsForYear :many
SELECT
    id,
    event_date,
    day_type_id,
    COALESCE(description, '') as description,
    created_at,
    updated_at
FROM calendar_events
WHERE YEAR(event_date) = YEAR(sqlc.arg(year))
ORDER BY event_date;

-- name: GetCalendarEventsById :one
SELECT
    id,
    event_date,
    day_type_id,
    COALESCE(description, '') as description,
    created_at,
    updated_at
FROM calendar_events
WHERE id = ?;

-- name: GetCalendarEventsByDate :one
SELECT
    id,
    event_date,
    day_type_id,
    COALESCE(description, '') as description,
    created_at,
    updated_at
FROM calendar_events
WHERE event_date = ?;

-- name: CreateCalendarEvents :execresult
INSERT INTO calendar_events (event_date, day_type_id, description)
VALUES (?, ?, ?);

-- name: UpdateCalendarEvents :exec
UPDATE calendar_events
SET
    event_date = ?,
    day_type_id = ?,
    description = ?
WHERE id = ?;

-- name: DeleteCalendarEvents :exec
DELETE FROM calendar_events WHERE id = ?;
