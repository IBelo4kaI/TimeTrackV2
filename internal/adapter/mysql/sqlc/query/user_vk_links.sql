-- name: LinkUserVK :exec
INSERT INTO
  user_vk_links (user_id, vk_user_id, created_at)
VALUES
  (?, ?, UTC_TIMESTAMP()) ON DUPLICATE KEY
UPDATE vk_user_id =
VALUES
  (vk_user_id);

-- name: UnlinkUserVK :exec
DELETE FROM user_vk_links
WHERE
  user_id = ?;

-- name: GetVKIDByUser :one
SELECT
  vk_user_id
FROM
  user_vk_links
WHERE
  user_id = ?;

-- name: GetUserByVKID :one
SELECT
  user_id
FROM
  user_vk_links
WHERE
  vk_user_id = ?;

-- name: ListVKIDsByUsers :many
-- Пакетно для рассылки уведомлений участникам чата одним запросом вместо
-- N+1 (по аналогии с ListFilesByEntityIDs в file_entity_refs.sql).
SELECT
  user_id,
  vk_user_id
FROM
  user_vk_links
WHERE
  user_id IN (sqlc.slice ('user_ids'));
