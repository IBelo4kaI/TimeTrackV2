-- Расширяем чек до полного набора реквизитов стандартного кассового чека
-- (см. обсуждение вывода чека на фронте) — раньше сохраняли только часть
-- полей, которые реально отдаёт "Проверка чека онлайн".
--
-- retail_place — это "Место расчётов" (например, "ТЦ 1357"), сейчас
-- отдельно не хранится и по ошибке используется на фронте как имя
-- продавца (см. mapExternalReceipt) — это отдельно чинится на фронте,
-- здесь только добавляем для него колонку.
--
-- nds22 — по аналогии с nds20/nds10/nds0/nds_no: новая ставка НДС 22%,
-- появившаяся в ответах сервиса в отдельной структуре amountsReceiptNds
-- (код ставки 11), а не в старых плоских полях nds20/nds10.
ALTER TABLE `receipts`
  ADD COLUMN `shift_number` INT DEFAULT NULL AFTER `taxation_type`,
  ADD COLUMN `kkt_reg_id` VARCHAR(20) DEFAULT NULL AFTER `shift_number`,
  ADD COLUMN `fiscal_document_format_ver` INT DEFAULT NULL AFTER `kkt_reg_id`,
  ADD COLUMN `machine_number` VARCHAR(50) DEFAULT NULL AFTER `fiscal_document_format_ver`,
  ADD COLUMN `retail_place` VARCHAR(255) DEFAULT NULL AFTER `machine_number`,
  ADD COLUMN `operator` VARCHAR(255) DEFAULT NULL AFTER `retail_place`,
  ADD COLUMN `prepaid_sum` BIGINT DEFAULT NULL AFTER `operator`,
  ADD COLUMN `nds22` BIGINT NOT NULL DEFAULT '0' AFTER `nds_no`;

-- raw-код ставки НДС по ФФД (тег 1199) на конкретную позицию чека —
-- 1/2/3/4/5/6 по спецификации + 11 (эмпирически подтверждено = 22%,
-- см. getNdsRateLabel); нужен, чтобы показать "НДС N%" под каждым товаром,
-- как на настоящем чеке.
ALTER TABLE `receipt_items`
  ADD COLUMN `nds_code` INT DEFAULT NULL AFTER `sum`;
