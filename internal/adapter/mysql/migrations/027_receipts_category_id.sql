-- Категория чека — проставляется автоматически при сохранении (см.
-- internal/receipt_category.ClassifyReceipt), сотрудник может поправить
-- вручную (PUT /receipts/:id/category). NULL — "Без категории" (ни ИНН
-- продавца, ни позиции чека ни с чем не совпали).
ALTER TABLE `receipts`
  ADD COLUMN `category_id` INT DEFAULT NULL AFTER `has_paper`;

ALTER TABLE `receipts`
  ADD CONSTRAINT `fk_receipts_category` FOREIGN KEY (`category_id`) REFERENCES `categories` (`id`) ON DELETE SET NULL;
