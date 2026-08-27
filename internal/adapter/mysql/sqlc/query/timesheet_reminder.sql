-- name: ListKnownUserIDs :many
-- Сотрудники, о которых бэк вообще что-то знает локально (без похода в
-- auth-сервис за полным списком — см. internal/timesheetreminder) — кто
-- хоть раз вносил запись в табель, либо кому задан индивидуальный график.
-- Ограничение: сотрудник, который вообще ничего ни разу не вносил, сюда не
-- попадёт — и не получит напоминание, хотя ему оно нужнее всего.
SELECT
  user_id
FROM
  user_time_entries
GROUP BY
  user_id
UNION
SELECT
  user_id
FROM
  work_standards
WHERE
  user_id IS NOT NULL
GROUP BY
  user_id;

-- name: CountNotificationsSentToday :one
-- Дедуп: не слать напоминание повторно в тот же день (DATE() по UTC —
-- created_at теперь буквальный UTC, см. 018_notification_timestamp_utc.sql).
SELECT
  COUNT(*)
FROM
  notifications
WHERE
  user_id = ?
  AND entity_type = ?
  AND entity_id = ?
  AND DATE(created_at) = DATE(UTC_TIMESTAMP());
