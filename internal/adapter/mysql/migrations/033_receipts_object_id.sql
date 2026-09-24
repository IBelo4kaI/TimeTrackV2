-- Объект (стройплощадка/проект) из Reference Service, на который потрачены
-- деньги по чеку. Храним только id — название живёт в справочнике, копию не
-- держим, чтобы не расходиться при переименовании. NULL — объект не указан.
ALTER TABLE `receipts`
  ADD COLUMN `object_id` VARCHAR(36) DEFAULT NULL AFTER `category_id`,
  ADD KEY `idx_receipts_object` (`object_id`);
