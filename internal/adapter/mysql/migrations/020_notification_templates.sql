-- Готовые шаблоны для ручной рассылки уведомлений админом сотрудникам (см.
-- internal/notification_template, internal/notification.Service.SendManual).
-- DATETIME + явный UTC_TIMESTAMP() в запросах, а не TIMESTAMP DEFAULT
-- CURRENT_TIMESTAMP — тот же баг, что уже чинили для чата и notifications
-- (013_chat_timestamps_utc.sql, 018_notification_timestamp_utc.sql): TIMESTAMP
-- пересчитывается через сессионную time_zone на каждой записи/чтении.
CREATE TABLE `notification_templates` (
  `id` VARCHAR(36) NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `title` VARCHAR(255) NOT NULL,
  `message` TEXT NOT NULL,
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_notification_templates_name` (`name`)
);
