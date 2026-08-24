-- ============================================
-- system_settings queries
-- ============================================
-- name: GetSystemSettings :many
SELECT
  *
FROM
  system_settings;

-- name: GetSystemSettingByKey :one
SELECT
  *
FROM
  system_settings
WHERE
  setting_key = ?;

-- name: GetSystemSettingByKeyAndCategory :one
SELECT
  *
FROM
  system_settings
WHERE
  setting_key = ?
  AND category = ?;

-- name: CreateSystemSetting :exec
INSERT INTO
  system_settings (
    setting_key,
    setting_value,
    setting_type,
    category,
    description,
    is_public
  )
VALUES
  (?, ?, ?, ?, ?, ?);

-- name: UpdateSystemSetting :exec
UPDATE system_settings
SET
  setting_value = ?,
  setting_type = ?,
  category = ?,
  description = ?,
  is_public = ?,
  updated_at = CURRENT_TIMESTAMP
WHERE
  setting_key = ?;

-- name: UpdateValueSystemSetting :exec
-- INSERT ... ON DUPLICATE, а не голый UPDATE: та настройка, которую ещё ни
-- разу не сохраняли (нет строки в system_settings — например, только что
-- заведённый ключ), голым UPDATE молча не создаётся (0 affected rows, без
-- ошибки) — значение просто терялось бы. setting_type/category берут
-- дефолт колонки ('string'/'general'), для JSON-настроек тип выставляется
-- отдельно через CreateSystemSetting/сид-миграцию.
INSERT INTO
  system_settings (setting_key, setting_value)
VALUES
  (?, ?) ON DUPLICATE KEY
UPDATE setting_value =
VALUES
  (setting_value);

-- name: DeleteSystemSetting :exec
DELETE FROM system_settings
WHERE
  setting_key = ?;

-- name: GetSystemSettingsByCategory :many
SELECT
  *
FROM
  system_settings
WHERE
  category = ?;

-- name: GetPublicSystemSettings :many
SELECT
  *
FROM
  system_settings
WHERE
  is_public = 1;
