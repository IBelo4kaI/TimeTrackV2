-- name: CreateNotificationTemplate :exec
INSERT INTO
  notification_templates (id, name, title, message, created_at, updated_at)
VALUES
  (?, ?, ?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP());

-- name: UpdateNotificationTemplate :exec
UPDATE notification_templates
SET
  name = ?,
  title = ?,
  message = ?,
  updated_at = UTC_TIMESTAMP()
WHERE
  id = ?;

-- name: DeleteNotificationTemplate :exec
DELETE FROM notification_templates
WHERE
  id = ?;

-- name: GetNotificationTemplateByID :one
SELECT
  *
FROM
  notification_templates
WHERE
  id = ?;

-- name: GetNotificationTemplateByName :one
SELECT
  *
FROM
  notification_templates
WHERE
  name = ?;

-- name: ListNotificationTemplates :many
SELECT
  *
FROM
  notification_templates
ORDER BY
  name;
