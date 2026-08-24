-- name: CreateNotification :exec
INSERT INTO
  notifications (user_id, title, message, type, entity_type, entity_id)
VALUES
  (?, ?, ?, ?, ?, ?);

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
