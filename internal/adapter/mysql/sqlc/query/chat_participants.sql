-- ============================================
-- chat_participants queries
-- ============================================
-- name: AddChatParticipant :exec
INSERT INTO
  chat_participants (chat_id, user_id, role, joined_at)
VALUES
  (?, ?, ?, UTC_TIMESTAMP());

-- name: RemoveChatParticipant :exec
DELETE FROM chat_participants
WHERE
  chat_id = ?
  AND user_id = ?;

-- name: GetChatParticipant :one
SELECT
  *
FROM
  chat_participants
WHERE
  chat_id = ?
  AND user_id = ?;

-- name: ListChatParticipants :many
SELECT
  *
FROM
  chat_participants
WHERE
  chat_id = ?
ORDER BY
  joined_at;

-- name: UpdateChatParticipantRole :exec
UPDATE chat_participants
SET
  role = ?
WHERE
  chat_id = ?
  AND user_id = ?;

-- name: SetChatParticipantMuted :exec
UPDATE chat_participants
SET
  muted = ?
WHERE
  chat_id = ?
  AND user_id = ?;

-- name: MarkChatRead :exec
UPDATE chat_participants
SET
  last_read_message_id = ?,
  last_read_at = UTC_TIMESTAMP()
WHERE
  chat_id = ?
  AND user_id = ?;

-- name: CountUnreadChatMessages :one
SELECT
  COUNT(*)
FROM
  chat_messages m
  INNER JOIN chat_participants cp ON cp.chat_id = m.chat_id
WHERE
  cp.chat_id = ?
  AND cp.user_id = ?
  AND m.is_deleted = FALSE
  AND m.id > COALESCE(cp.last_read_message_id, 0);
