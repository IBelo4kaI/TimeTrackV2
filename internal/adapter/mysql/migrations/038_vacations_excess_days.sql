-- Превышение доступного остатка на момент создания заявки.
-- Для старых заявок исторический остаток неизвестен: оставляем 0.
ALTER TABLE vacations
  ADD COLUMN excess_days INT NOT NULL DEFAULT 0 AFTER total_days;
