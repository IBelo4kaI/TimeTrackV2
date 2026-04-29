-- name: CreateFileEntityRef :exec
INSERT INTO file_entity_refs (file_id, entity_type, entity_id)
VALUES (?, ?, ?);

-- name: ListFilesByEntity :many
SELECT f.id, f.original_name, f.storage_path, f.mime_type, f.file_type, f.size_bytes, f.checksum,
       f.uploaded_by_user_id, f.is_deleted, f.deleted_at, f.created_at, f.updated_at
FROM files f
INNER JOIN file_entity_refs r ON r.file_id = f.id
WHERE r.entity_type = ? AND r.entity_id = ? AND f.is_deleted = FALSE
ORDER BY r.created_at DESC;

-- name: DeleteAllFileEntityRefsByFile :exec
DELETE FROM file_entity_refs WHERE file_id = ?;
