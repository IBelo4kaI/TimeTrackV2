-- name: CreateNotification :exec
INSERT INTO
  notifications (id, user_id, title, message, type, entity_type, entity_id, created_at)
VALUES
  (?, ?, ?, ?, ?, ?, ?, UTC_TIMESTAMP());

-- name: ListNotificationsByUser :many
SELECT
  *
FROM
  notifications
WHERE
  user_id = ?
ORDER BY
  created_at DESC
LIMIT
  ?
OFFSET
  ?;

-- name: CountUnreadNotifications :one
SELECT
  COUNT(*)
FROM
  notifications
WHERE
  user_id = ?
  AND is_read = FALSE;

-- name: MarkNotificationRead :exec
UPDATE notifications
SET
  is_read = TRUE
WHERE
  id = ?
  AND user_id = ?;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications
SET
  is_read = TRUE
WHERE
  user_id = ?
  AND is_read = FALSE;

-- name: MarkNotificationsReadByEntity :exec
-- Прочитать разом все уведомления по сущности (например, все накопленные
-- уведомления о новых сообщениях в чате — при открытии/прочтении чата).
UPDATE notifications
SET
  is_read = TRUE
WHERE
  user_id = ?
  AND entity_type = ?
  AND entity_id = ?
  AND is_read = FALSE;

-- name: DeleteNotification :exec
DELETE FROM notifications
WHERE
  id = ?
  AND user_id = ?;

-- name: DeleteAllNotificationsByUser :exec
DELETE FROM notifications
WHERE
  user_id = ?;
