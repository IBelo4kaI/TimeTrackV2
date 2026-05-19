-- name: CreateFile :exec
INSERT INTO
  files (
    id,
    original_name,
    storage_path,
    mime_type,
    file_type,
    size_bytes,
    checksum,
    uploaded_by_user_id
  )
VALUES
  (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetFileByID :one
SELECT
  id,
  original_name,
  storage_path,
  mime_type,
  file_type,
  size_bytes,
  checksum,
  uploaded_by_user_id,
  is_deleted,
  deleted_at,
  created_at,
  updated_at
FROM
  files
WHERE
  id = ?
  AND is_deleted = FALSE;

-- name: ListFilesByUploader :many
SELECT
  id,
  original_name,
  storage_path,
  mime_type,
  file_type,
  size_bytes,
  checksum,
  uploaded_by_user_id,
  is_deleted,
  deleted_at,
  created_at,
  updated_at
FROM
  files
WHERE
  uploaded_by_user_id = ?
  AND is_deleted = FALSE
ORDER BY
  created_at DESC;

-- name: SoftDeleteFile :exec
UPDATE files
SET
  is_deleted = TRUE,
  deleted_at = CURRENT_TIMESTAMP
WHERE
  id = ?
  AND is_deleted = FALSE;

-- name: HardDeleteFile :exec
DELETE FROM files
WHERE
  id = ?
