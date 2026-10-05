-- name: CreateFileEntityRef :exec
INSERT INTO
  file_entity_refs (file_id, entity_type, entity_id)
VALUES
  (?, ?, ?);

-- name: ListFilesByEntity :many
SELECT
  f.id,
  f.original_name,
  f.storage_path,
  f.mime_type,
  f.file_type,
  f.category_id,
  f.size_bytes,
  f.checksum,
  f.uploaded_by_user_id,
  f.is_deleted,
  f.deleted_at,
  f.created_at,
  f.updated_at
FROM
  files f
  INNER JOIN file_entity_refs r ON r.file_id = f.id
WHERE
  r.entity_type = ?
  AND r.entity_id = ?
  AND f.is_deleted = FALSE
  AND (sqlc.narg (year) IS NULL OR YEAR(f.created_at) = CAST(sqlc.narg (year) AS SIGNED))
ORDER BY
  r.created_at DESC;

-- name: ListFilesByEntityType :many
SELECT
  f.id,
  f.original_name,
  f.storage_path,
  f.mime_type,
  f.file_type,
  f.category_id,
  f.size_bytes,
  f.checksum,
  f.uploaded_by_user_id,
  f.is_deleted,
  f.deleted_at,
  f.created_at,
  f.updated_at,
  r.entity_type,
  r.entity_id
FROM
  files f
  INNER JOIN file_entity_refs r ON r.file_id = f.id
WHERE
  r.entity_type = ?
  AND f.is_deleted = FALSE
  AND (sqlc.narg (year) IS NULL OR YEAR(f.created_at) = CAST(sqlc.narg (year) AS SIGNED))
ORDER BY
  r.created_at DESC;

-- name: DeleteAllFileEntityRefsByFile :exec
DELETE FROM file_entity_refs
WHERE
  file_id = ?;

-- name: ListFilesByEntityIDs :many
-- Вложения сразу для НЕСКОЛЬКИХ сущностей одного типа за один запрос —
-- нужно, чтобы отдать список сообщений чата со вложениями без N+1
-- (см. internal/chat/service.go, attachmentsForMessages).
SELECT
  f.id,
  f.original_name,
  f.mime_type,
  f.file_type,
  f.size_bytes,
  r.entity_id
FROM
  files f
  INNER JOIN file_entity_refs r ON r.file_id = f.id
WHERE
  r.entity_type = ?
  AND r.entity_id IN (sqlc.slice ('entity_ids'))
  AND f.is_deleted = FALSE
ORDER BY
  r.created_at;

-- name: ListVacationIDsByUser :many
SELECT
  id
FROM
  vacations
WHERE
  user_id = ?;

-- name: ListReceiptIDsByUser :many
SELECT
  id
FROM
  receipts
WHERE
  user_id = ?;

-- name: ListSickLeaveIDsByUser :many
SELECT
  id
FROM
  sick_leaves
WHERE
  user_id = ?;

-- name: ListEntityRefsByFile :many
SELECT
  entity_type,
  entity_id
FROM
  file_entity_refs
WHERE
  file_id = ?;

-- name: GetVacationOwnerID :one
SELECT
  user_id
FROM
  vacations
WHERE
  id = ?;

-- name: GetReceiptOwnerID :one
SELECT
  user_id
FROM
  receipts
WHERE
  id = ?;

-- name: GetSickLeaveOwnerID :one
SELECT
  user_id
FROM
  sick_leaves
WHERE
  id = ?;

-- name: IsChatMessageParticipant :one
SELECT
  EXISTS (
    SELECT
      1
    FROM
      chat_messages m
      INNER JOIN chat_participants p ON p.chat_id = m.chat_id
    WHERE
      m.id = ?
      AND p.user_id = ?
  ) AS is_participant;
