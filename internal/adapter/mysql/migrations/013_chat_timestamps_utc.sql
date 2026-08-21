-- TIMESTAMP-колонки MySQL пересчитывают значение через сессионную time_zone
-- на каждой записи/чтении — DATETIME хранит буквально, без пересчёта.
-- Переводим на DATETIME; created_at/joined_at теперь без DEFAULT — их
-- значение (UTC_TIMESTAMP()) явно передаётся из запросов (см. chats.sql,
-- chat_messages.sql, chat_participants.sql), а не выставляется MySQL.
--
-- chats.updated_at не трогаем: ON UPDATE не принимает произвольные
-- выражения (только CURRENT_TIMESTAMP), а поле нигде не используется.

ALTER TABLE `chats`
  MODIFY COLUMN `last_message_at` datetime NULL DEFAULT NULL,
  MODIFY COLUMN `created_at` datetime NOT NULL;

ALTER TABLE `chat_messages`
  MODIFY COLUMN `deleted_at` datetime NULL DEFAULT NULL,
  MODIFY COLUMN `created_at` datetime NOT NULL;

ALTER TABLE `chat_participants`
  MODIFY COLUMN `last_read_at` datetime NULL DEFAULT NULL,
  MODIFY COLUMN `joined_at` datetime NOT NULL;
