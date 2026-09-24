-- Отдельно от 034: ALTER'ы не идемпотентны, а 034 должна уметь перезапускаться.
ALTER TABLE `receipts` ADD COLUMN `categories_manual` TINYINT(1) NOT NULL DEFAULT 0 AFTER `has_paper`;

ALTER TABLE `receipts` DROP FOREIGN KEY `fk_receipts_category`;

ALTER TABLE `receipts` DROP COLUMN `category_id`;
