-- Ручной ввод чека (см. internal/receipt) — без сканирования QR нет ни
-- фискальных реквизитов, ни гарантированного ИНН продавца. NULL, а не
-- пустая строка: у UNIQUE KEY uq_receipts_fiscal MySQL не считает несколько
-- NULL дубликатами друг друга (в отличие от нескольких пустых строк),
-- так что ручные чеки не будут ложно конфликтовать между собой.
ALTER TABLE `receipts`
  MODIFY COLUMN `fiscal_drive_number` VARCHAR(20) DEFAULT NULL,
  MODIFY COLUMN `fiscal_document_number` VARCHAR(20) DEFAULT NULL,
  MODIFY COLUMN `fiscal_sign` VARCHAR(20) DEFAULT NULL,
  MODIFY COLUMN `seller_inn` VARCHAR(12) DEFAULT NULL;
