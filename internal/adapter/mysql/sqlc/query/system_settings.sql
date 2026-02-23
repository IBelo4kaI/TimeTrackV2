-- ============================================
-- system_settings queries
-- ============================================

-- name: GetSystemSettings :many
SELECT * FROM system_settings;

-- name: GetSystemSettingByKey :one
SELECT * FROM system_settings WHERE setting_key = ?;

-- name: GetSystemSettingByKeyAndCategory :one
SELECT * FROM system_settings WHERE setting_key = ? AND category = ?;

-- name: CreateSystemSetting :exec
INSERT INTO system_settings (setting_key, setting_value, setting_type, category, description, is_public)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateSystemSetting :exec
UPDATE system_settings
SET setting_value = ?, setting_type = ?, category = ?, description = ?, is_public = ?, updated_at = CURRENT_TIMESTAMP
WHERE setting_key = ?;

-- name: UpdateValueSystemSetting :exec
UPDATE system_settings
SET setting_value = ?
WHERE setting_key = ?;

-- name: DeleteSystemSetting :exec
DELETE FROM system_settings WHERE setting_key = ?;

-- name: GetSystemSettingsByCategory :many
SELECT * FROM system_settings WHERE category = ?;

-- name: GetPublicSystemSettings :many
SELECT * FROM system_settings WHERE is_public = 1;
