-- name: CreateNewsPost :exec
INSERT INTO
  news_posts (id, title, body, created_at, updated_at)
VALUES
  (?, ?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP());

-- name: UpdateNewsPost :exec
UPDATE news_posts
SET
  title = ?,
  body = ?,
  updated_at = UTC_TIMESTAMP()
WHERE
  id = ?;

-- name: DeleteNewsPost :exec
DELETE FROM news_posts
WHERE
  id = ?;

-- name: GetNewsPostByID :one
SELECT
  *
FROM
  news_posts
WHERE
  id = ?;

-- name: ListNewsPosts :many
SELECT
  *
FROM
  news_posts
ORDER BY
  created_at DESC;

-- name: CountNewsPostsSince :one
SELECT
  COUNT(*)
FROM
  news_posts
WHERE
  created_at > ?;
