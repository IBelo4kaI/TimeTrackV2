-- Категории файлов с вложенностью (дерево папок) + привязка файла к категории.

CREATE TABLE IF NOT EXISTS `file_categories` (
  `id` varchar(36) NOT NULL DEFAULT (uuid()),
  `name` varchar(100) NOT NULL,
  `parent_id` varchar(36) DEFAULT NULL,
  `color_code` varchar(7) DEFAULT NULL,
  `sort_order` int NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_file_categories_parent` (`parent_id`),
  CONSTRAINT `fk_file_categories_parent` FOREIGN KEY (`parent_id`) REFERENCES `file_categories` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE `files`
  ADD COLUMN `category_id` varchar(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL AFTER `file_type`,
  ADD KEY `idx_files_category` (`category_id`),
  ADD CONSTRAINT `fk_files_category` FOREIGN KEY (`category_id`) REFERENCES `file_categories` (`id`) ON DELETE SET NULL ON UPDATE CASCADE;
