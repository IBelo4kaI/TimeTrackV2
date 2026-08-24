-- Отдельно от muted (глушит всё — тост/браузер/звук/VK) — vk_muted глушит
-- только VK-дубликат, приложение продолжает уведомлять как обычно.
ALTER TABLE `chat_participants`
  ADD COLUMN `vk_muted` tinyint(1) NOT NULL DEFAULT 0 AFTER `muted`;
