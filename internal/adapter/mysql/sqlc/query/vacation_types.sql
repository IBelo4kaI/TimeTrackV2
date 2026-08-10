-- ============================================
-- vacation_types queries
-- ============================================
-- name: GetVacationTypes :many
SELECT
  *
FROM
  vacation_types
ORDER BY
  sort_order,
  name;

-- name: GetActiveVacationTypes :many
SELECT
  *
FROM
  vacation_types
WHERE
  is_active = TRUE
ORDER BY
  sort_order,
  name;

-- name: GetVacationTypeByID :one
SELECT
  *
FROM
  vacation_types
WHERE
  id = ?;

-- name: GetVacationTypeBySystemName :one
SELECT
  *
FROM
  vacation_types
WHERE
  system_name = ?;

-- name: CreateVacationType :exec
INSERT INTO
  vacation_types (
    id,
    name,
    system_name,
    color_code,
    affects_balance,
    is_active,
    sort_order
  )
VALUES
  (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateVacationType :exec
UPDATE vacation_types
SET
  name = ?,
  system_name = ?,
  color_code = ?,
  affects_balance = ?,
  is_active = ?,
  sort_order = ?
WHERE
  id = ?;

-- name: DeleteVacationType :exec
DELETE FROM vacation_types
WHERE
  id = ?;

-- name: CountVacationsByType :one
SELECT
  COUNT(*)
FROM
  vacations
WHERE
  vacation_type_id = ?;
