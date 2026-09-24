-- Несколько категорий на чек ("многие ко многим"): связь вынесена из
-- receipts.category_id в receipt_categories. categories_manual — категории
-- выбраны вручную: автоклассификация (Backfill, обновление по продавцу) такой
-- чек не трогает. Существующие категории копируются как автоматические.
-- Идемпотентно (IF NOT EXISTS/IGNORE): DDL в MySQL коммитится сам, и
-- миграция, упавшая посередине, должна безопасно перезапускаться.
CREATE TABLE IF NOT EXISTS `receipt_categories` (
  `receipt_id` VARCHAR(36) NOT NULL,
  `category_id` INT NOT NULL,
  PRIMARY KEY (`receipt_id`, `category_id`),
  KEY `idx_receipt_categories_category` (`category_id`),
  CONSTRAINT `fk_receipt_categories_receipt` FOREIGN KEY (`receipt_id`) REFERENCES `receipts` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_receipt_categories_category` FOREIGN KEY (`category_id`) REFERENCES `categories` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `receipt_categories` (`receipt_id`, `category_id`)
SELECT `id`, `category_id` FROM `receipts` WHERE `category_id` IS NOT NULL;
