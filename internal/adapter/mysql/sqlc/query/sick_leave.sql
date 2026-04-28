-- name: GetSickLeaveByID :one
SELECT
    id,
    user_id,
    start_date,
    end_date,
    total_days,
    COALESCE(description, '') as description,
    COALESCE(doc_file_name, '') as doc_file_name,
    status,
    created_at,
    updated_at
FROM sick_leaves
WHERE id = ?;

-- name: UpdateSickLeaveStatus :exec
UPDATE sick_leaves
SET status = ?
WHERE id = ?;

-- name: DeleteSickLeave :exec
DELETE FROM sick_leaves WHERE id = ?;

-- name: GetSickLeavesByYear :many
SELECT
    id,
    user_id,
    start_date,
    end_date,
    total_days,
    COALESCE(description, '') as description,
    COALESCE(doc_file_name, '') as doc_file_name,
    status,
    created_at,
    updated_at
FROM sick_leaves
WHERE user_id = sqlc.arg(user_id)
    AND YEAR(start_date) = YEAR(sqlc.arg(year))
    AND YEAR(end_date) = YEAR(sqlc.arg(year))
ORDER BY created_at DESC;

-- name: GetAllUsersSickLeavesByYear :many
SELECT
    id,
    user_id,
    start_date,
    end_date,
    total_days,
    COALESCE(description, '') as description,
    COALESCE(doc_file_name, '') as doc_file_name,
    status,
    created_at,
    updated_at
FROM sick_leaves
WHERE YEAR(start_date) = YEAR(sqlc.arg(year))
    AND YEAR(end_date) = YEAR(sqlc.arg(year))
ORDER BY created_at DESC;

-- name: CreateSickLeave :exec
INSERT INTO sick_leaves (user_id, start_date, end_date, total_days, description, doc_file_name, status)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetCountSickLeavesByStatus :one
SELECT COALESCE(SUM(total_days), 0) as total_days
FROM sick_leaves
WHERE user_id = ?
  AND status = ?
  AND YEAR(start_date) = YEAR(sqlc.arg(year));

-- name: UpdateSickLeaveFileName :exec
UPDATE sick_leaves
SET doc_file_name = ?
WHERE id = ?;
