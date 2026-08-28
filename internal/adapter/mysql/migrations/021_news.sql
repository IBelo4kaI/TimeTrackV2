-- Новости/чейнджлог приложения — админ публикует посты (see internal/news),
-- сотрудники видят бейдж непрочитанного + попап/модалку "что нового".
-- DATETIME + явный UTC_TIMESTAMP() в запросах — та же схема, что и для
-- notification_templates (020_notification_templates.sql), чтобы не словить
-- баг с пересчётом TIMESTAMP через сессионную time_zone.
CREATE TABLE `news_posts` (
  `id` VARCHAR(36) NOT NULL,
  `title` VARCHAR(255) NOT NULL,
  `body` TEXT NOT NULL,
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL,
  PRIMARY KEY (`id`)
);

-- Одна строка на пользователя (не на пост, в отличие от notifications) —
-- новости общие для всех, отслеживаем только "видел до какого момента".
CREATE TABLE `news_read_marks` (
  `user_id` VARCHAR(36) NOT NULL,
  `last_seen_at` DATETIME NOT NULL,
  PRIMARY KEY (`user_id`)
);
