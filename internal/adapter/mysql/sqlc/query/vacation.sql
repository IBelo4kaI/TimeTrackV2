-- name: GetVacationByID :one
SELECT
  v.id,
  v.user_id,
  v.start_date,
  v.end_date,
  v.total_days,
  v.excess_days,
  v.available_days,
  v.decision_comment,
  COALESCE(v.description, '') as description,
  v.status,
  v.approval_email_sent_at,
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

-- name: MarkVacationApprovalEmailSent :exec
UPDATE vacations
SET
  approval_email_sent_at = ?
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
  v.excess_days,
  v.available_days,
  v.decision_comment,
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
  AND YEAR(v.start_date) <= YEAR(sqlc.arg (year))
  AND YEAR(v.end_date) >= YEAR(sqlc.arg (year))
ORDER BY
  v.created_at DESC;

-- name: GetAllUsersVacationsByYear :many
SELECT
  v.id,
  v.user_id,
  v.start_date,
  v.end_date,
  v.total_days,
  v.excess_days,
  v.available_days,
  v.decision_comment,
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
  YEAR(v.start_date) <= YEAR(sqlc.arg (year))
  AND YEAR(v.end_date) >= YEAR(sqlc.arg (year))
ORDER BY
  v.created_at DESC;

-- name: ListVacationCalendarByYear :many
-- Урезанный набор полей отпусков ВСЕХ сотрудников для виджета "отпуска
-- коллег" (internal/vacation/service.go ListVacationCalendarByYear) — без
-- description: это личная причина отпуска, её не должен видеть весь
-- коллектив, в отличие от самого факта и дат отпуска. Отдельная ручка
-- (GET /vacation/calendar/:year) с отдельным разрешением
-- time:vacation_calendar:read — специально, чтобы НЕ приходилось выдавать
-- всем сотрудникам time:vacation.all:read (тот открывает куда более
-- чувствительный полный список + доступ к менеджерским действиям).
SELECT
  v.id,
  v.user_id,
  v.start_date,
  v.end_date,
  v.total_days,
  v.excess_days,
  v.status,
  v.vacation_type_id,
  COALESCE(t.name, '') as vacation_type_name,
  COALESCE(t.color_code, '') as vacation_type_color
FROM
  vacations v
  LEFT JOIN vacation_types t ON t.id = v.vacation_type_id
WHERE
  YEAR(v.start_date) <= YEAR(sqlc.arg (year))
  AND YEAR(v.end_date) >= YEAR(sqlc.arg (year))
ORDER BY
  v.created_at DESC;

-- name: CreateVacation :exec
-- id передаём явно (не полагаемся на DEFAULT(uuid())) — нужен сразу после
-- вставки, чтобы привязать к нему уведомления админам (см. internal/vacation
-- /service.go CreateVacationReport).
INSERT INTO
  vacations (
    id,
    user_id,
    start_date,
    end_date,
    total_days,
    excess_days,
    available_days,
    description,
    status,
    vacation_type_id
  )
VALUES
  (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

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

-- name: LockVacation :one
SELECT id, user_id, status FROM vacations WHERE id = ? FOR UPDATE;

-- name: SetVacationDecision :exec
UPDATE vacations SET status = ?, decision_comment = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: EditPendingVacation :exec
UPDATE vacations SET start_date = ?, end_date = ?, total_days = ?,
 vacation_type_id = ?, description = ?, excess_days = ?, available_days = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND status = 'pending';

-- name: AddVacationHistory :exec
INSERT INTO vacation_history (vacation_id, actor_id, event_type, detail) VALUES (?, ?, ?, ?);

-- name: ListVacationHistory :many
SELECT id, vacation_id, actor_id, event_type, detail, created_at FROM vacation_history
WHERE vacation_id = ? ORDER BY created_at DESC, id DESC;
