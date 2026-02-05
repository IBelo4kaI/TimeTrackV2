-- ============================================
-- work_standards queries
-- ============================================

-- name: GetWorkStandardsById :one
SELECT *
FROM work_standards
WHERE id = ?;

-- name: GetWorkStandardsByMonthAndGenderId :one
SELECT *
FROM work_standards
WHERE month = ? AND year = ? AND gender = ? AND (user_id IS NULL OR user_id = ?);

-- name: GetWorkStandardsByMonth :many
SELECT *
FROM work_standards
WHERE month = ? AND year = ?
ORDER BY gender, user_id;

-- name: GetWorkStandardsByYear :many
SELECT *
FROM work_standards
WHERE year = ?
ORDER BY month, gender, user_id;

-- name: CreateWorkStandard :exec
INSERT INTO work_standards (user_id, month, year, standard_hours, standard_days, gender)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateWorkStandard :exec
UPDATE work_standards
SET
    standard_hours = ?,
    standard_days = ?
WHERE id = ?;

-- name: DeleteWorkStandard :exec
DELETE FROM work_standards WHERE id = ?;
