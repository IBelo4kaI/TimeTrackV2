-- Тот же баг и то же лекарство, что в 013_chat_timestamps_utc.sql: MySQL
-- TIMESTAMP пересчитывает значение через сессионную time_zone на каждой
-- записи/чтении, даже если писали UTC_TIMESTAMP() — DATETIME хранит буквально.
-- created_at теперь без DEFAULT — значение (UTC_TIMESTAMP()) явно передаётся
-- из CreateNotification (см. query/notifications.sql), а не выставляется MySQL.
ALTER TABLE `notifications`
  MODIFY COLUMN `created_at` datetime NOT NULL;
