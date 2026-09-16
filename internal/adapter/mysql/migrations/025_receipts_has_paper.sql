-- Отметка "есть бумажный экземпляр чека" — сервис "Проверка чека онлайн"
-- этого не знает, отмечает сам сотрудник при добавлении чека (галочка на
-- фронте, см. ReceiptScan.vue). TINYINT(1) — реальный флаг, не код, sqlc
-- смаппит его в bool (см. комментарий про operation_type/taxation_type в
-- 022_add_receipts.sql про обратный случай).
ALTER TABLE `receipts`
  ADD COLUMN `has_paper` TINYINT(1) NOT NULL DEFAULT 0 AFTER `operation_type`;
