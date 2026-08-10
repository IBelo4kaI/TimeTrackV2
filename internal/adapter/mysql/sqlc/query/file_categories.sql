-- ============================================
-- file_categories queries
-- ============================================
-- name: GetFileCategories :many
SELECT
  *
FROM
  file_categories
ORDER BY
  sort_order,
  name;

-- name: GetFileCategoryByID :one
SELECT
  *
FROM
  file_categories
WHERE
  id = ?;

-- name: GetFileCategoryByParentAndName :one
SELECT
  *
FROM
  file_categories
WHERE
  name = ?
  AND parent_id <=> ?;

-- name: GetFileCategoryBySystemName :one
SELECT
  *
FROM
  file_categories
WHERE
  system_name = ?;

-- name: CountFileCategoryChildren :one
SELECT
  COUNT(*)
FROM
  file_categories
WHERE
  parent_id = ?;

-- name: CountFilesInCategory :one
SELECT
  COUNT(*)
FROM
  files
WHERE
  category_id = ?
  AND is_deleted = FALSE;

-- name: CreateFileCategory :exec
INSERT INTO
  file_categories (id, name, parent_id, color_code, sort_order)
VALUES
  (?, ?, ?, ?, ?);

-- name: UpdateFileCategory :exec
UPDATE file_categories
SET
  name = ?,
  parent_id = ?,
  color_code = ?,
  sort_order = ?
WHERE
  id = ?;

-- name: DeleteFileCategory :exec
DELETE FROM file_categories
WHERE
  id = ?;
