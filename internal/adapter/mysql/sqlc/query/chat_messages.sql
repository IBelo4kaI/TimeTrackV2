-- ============================================
-- chat_messages queries
-- ============================================
-- name: CreateChatMessage :execresult
INSERT INTO
  chat_messages (
    chat_id,
    sender_user_id,
    body,
    entity_type,
    entity_id,
    entity_title,
    entity_subtitle
  )
VALUES
  (?, ?, ?, ?, ?, ?, ?);

-- name: GetChatMessageByID :one
SELECT
  *
FROM
  chat_messages
WHERE
  id = ?;

-- name: ListChatMessages :many
SELECT
  *
FROM
  chat_messages
WHERE
  chat_id = ?
  AND is_deleted = FALSE
  AND (sqlc.narg (before_id) IS NULL OR id < sqlc.narg (before_id))
ORDER BY
  id DESC
LIMIT
  ?;

-- name: SoftDeleteChatMessage :exec
UPDATE chat_messages
SET
  is_deleted = TRUE,
  deleted_at = CURRENT_TIMESTAMP
WHERE
  id = ?;
