-- Отключение уведомлений по конкретному чату — персонально для каждого
-- участника (как role/last_read_at), не для чата в целом.
ALTER TABLE `chat_participants`
  ADD COLUMN `muted` tinyint(1) NOT NULL DEFAULT 0 AFTER `role`;
