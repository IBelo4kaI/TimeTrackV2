-- name: GetVacationByID :one
SELECT
    id,
    user_id,
    start_date,
    end_date,
    total_days,
    COALESCE(description, '') as description,
    status,
    created_at,
    updated_at
FROM vacations
WHERE id = ?;

-- name: UpdateVacationStatus :exec
UPDATE vacations
SET status = ?
WHERE id = ?;

-- name: DeleteVacation :exec
DELETE FROM vacations WHERE id = ?;

-- name: GetVacationsByYear :many
SELECT
    id,
    user_id,
    start_date,
    end_date,
    total_days,
    COALESCE(description, '') as description,
    status,
    created_at,
    updated_at
FROM vacations
WHERE user_id = sqlc.arg(user_id)
    AND YEAR(start_date) = YEAR(sqlc.arg(year))
    AND YEAR(end_date) = YEAR(sqlc.arg(year))
ORDER BY created_at DESC;

-- name: GetAllUsersVacationsByYear :many
SELECT
    id,
    user_id,
    start_date,
    end_date,
    total_days,
    COALESCE(description, '') as description,
    status,
    created_at,
    updated_at
FROM vacations
WHERE YEAR(start_date) = YEAR(sqlc.arg(year))
    AND YEAR(end_date) = YEAR(sqlc.arg(year))
ORDER BY created_at DESC;

-- name: CreateVacation :exec
INSERT INTO vacations (user_id, start_date, end_date, total_days, description, status)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetCountVacationsByStatus :one
SELECT COALESCE(SUM(total_days), 0) as total_days
FROM vacations
WHERE user_id = ?
  AND status = ?
  AND YEAR(start_date) = YEAR(sqlc.arg(year));
