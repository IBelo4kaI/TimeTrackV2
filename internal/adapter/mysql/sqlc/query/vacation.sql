-- name: GetVacationByID :one
SELECT
  v.id,
  v.user_id,
  v.start_date,
  v.end_date,
  v.total_days,
  COALESCE(v.description, '') as description,
  v.status,
  v.vacation_type_id,
  COALESCE(t.name, '') as vacation_type_name,
  COALESCE(t.color_code, '') as vacation_type_color,
  COALESCE(t.affects_balance, TRUE) as vacation_type_affects_balance,
  v.created_at,
  v.updated_at
FROM
  vacations v
  LEFT JOIN vacation_types t ON t.id = v.vacation_type_id
WHERE
  v.id = ?;

-- name: UpdateVacationStatus :exec
UPDATE vacations
SET
  status = ?
WHERE
  id = ?;

-- name: AssignVacationType :exec
UPDATE vacations
SET
  vacation_type_id = ?
WHERE
  id = ?;

-- name: DeleteVacation :exec
DELETE FROM vacations
WHERE
  id = ?;

-- name: GetVacationsByYear :many
SELECT
  v.id,
  v.user_id,
  v.start_date,
  v.end_date,
  v.total_days,
  COALESCE(v.description, '') as description,
  v.status,
  v.vacation_type_id,
  COALESCE(t.name, '') as vacation_type_name,
  COALESCE(t.color_code, '') as vacation_type_color,
  COALESCE(t.affects_balance, TRUE) as vacation_type_affects_balance,
  v.created_at,
  v.updated_at
FROM
  vacations v
  LEFT JOIN vacation_types t ON t.id = v.vacation_type_id
WHERE
  v.user_id = sqlc.arg (user_id)
  AND YEAR(v.start_date) = YEAR(sqlc.arg (year))
  AND YEAR(v.end_date) = YEAR(sqlc.arg (year))
ORDER BY
  v.created_at DESC;

-- name: GetAllUsersVacationsByYear :many
SELECT
  v.id,
  v.user_id,
  v.start_date,
  v.end_date,
  v.total_days,
  COALESCE(v.description, '') as description,
  v.status,
  v.vacation_type_id,
  COALESCE(t.name, '') as vacation_type_name,
  COALESCE(t.color_code, '') as vacation_type_color,
  COALESCE(t.affects_balance, TRUE) as vacation_type_affects_balance,
  v.created_at,
  v.updated_at
FROM
  vacations v
  LEFT JOIN vacation_types t ON t.id = v.vacation_type_id
WHERE
  YEAR(v.start_date) = YEAR(sqlc.arg (year))
  AND YEAR(v.end_date) = YEAR(sqlc.arg (year))
ORDER BY
  v.created_at DESC;

-- name: CreateVacation :exec
INSERT INTO
  vacations (
    user_id,
    start_date,
    end_date,
    total_days,
    description,
    status,
    vacation_type_id
  )
VALUES
  (?, ?, ?, ?, ?, ?, ?);

-- name: GetCountVacationsByStatus :one
SELECT
  COALESCE(SUM(v.total_days), 0) as total_days
FROM
  vacations v
  LEFT JOIN vacation_types t ON t.id = v.vacation_type_id
WHERE
  v.user_id = ?
  AND v.status = ?
  AND YEAR(v.start_date) = YEAR(sqlc.arg (year))
  AND COALESCE(t.affects_balance, TRUE) = TRUE;
