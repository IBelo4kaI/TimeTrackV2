-- ============================================
-- chats queries
-- ============================================
-- name: CreateChat :exec
INSERT INTO
  chats (id, type, name, entity_type, entity_id, created_by_user_id, created_at)
VALUES
  (?, ?, ?, ?, ?, ?, UTC_TIMESTAMP());

-- name: GetChatByID :one
SELECT
  *
FROM
  chats
WHERE
  id = ?;

-- name: GetChatByEntity :one
SELECT
  *
FROM
  chats
WHERE
  entity_type = ?
  AND entity_id = ?;

-- name: ListChatsByUser :many
SELECT
  c.id,
  c.type,
  c.name,
  c.entity_type,
  c.entity_id,
  c.created_by_user_id,
  c.last_message_at,
  c.created_at,
  c.updated_at,
  cp.role,
  cp.last_read_message_id,
  cp.last_read_at,
  cp.muted
FROM
  chats c
  INNER JOIN chat_participants cp ON cp.chat_id = c.id
WHERE
  cp.user_id = ?
ORDER BY
  c.last_message_at DESC,
  c.created_at DESC;

-- name: UpdateChatName :exec
UPDATE chats
SET
  name = ?
WHERE
  id = ?;

-- name: TouchChatLastMessage :exec
UPDATE chats
SET
  last_message_at = ?
WHERE
  id = ?;

-- name: DeleteChat :exec
-- Каскадно удаляет chat_participants и chat_messages (ON DELETE CASCADE,
-- см. миграцию 011_add_chats.sql).
DELETE FROM chats
WHERE
  id = ?;
