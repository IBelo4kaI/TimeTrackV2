-- name: GetNewsReadMark :one
SELECT
  *
FROM
  news_read_marks
WHERE
  user_id = ?;

-- name: UpsertNewsReadMark :exec
INSERT INTO
  news_read_marks (user_id, last_seen_at)
VALUES
  (?, UTC_TIMESTAMP())
ON DUPLICATE KEY UPDATE
  last_seen_at = UTC_TIMESTAMP();
