-- ============================================
-- day_types queries
-- ============================================

-- name: GetDayTypes :many
SELECT * FROM day_types;

-- name: GetDayTypeByID :one
SELECT * FROM day_types WHERE id = ?;

-- name: CreateDayType :exec
INSERT INTO day_types (name, system_name, is_work_day, affects_vacation, color_code) VALUES (?, ?, ?, ?, ?);

-- name: UpdateNameDayType :exec
UPDATE day_types SET name = ? WHERE id = ?;

-- name: UpdateSystemNameDayType :exec
UPDATE day_types SET system_name = ? WHERE id = ?;

-- name: UpdateIsWorkDayType :exec
UPDATE day_types SET is_work_day = ? WHERE id = ?;

-- name: UpdateAffectsVacationDayType :exec
UPDATE day_types SET affects_vacation = ? WHERE id = ?;

-- name: UpdateColorCodeDayType :exec
UPDATE day_types SET color_code = ? WHERE id = ?;

-- name: UpdateDayType :exec
UPDATE day_types SET name = ?, system_name = ?, is_work_day = ?, affects_vacation = ?, color_code = ? WHERE id = ?;

-- name: DeleteDayType :exec
DELETE FROM day_types WHERE id = ?;
