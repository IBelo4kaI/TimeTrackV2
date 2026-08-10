-- Типы отпусков (аналог day_types) + привязка отпуска к типу.

CREATE TABLE IF NOT EXISTS `vacation_types` (
  `id` varchar(36) NOT NULL DEFAULT (uuid()),
  `name` varchar(100) NOT NULL,
  `system_name` varchar(50) NOT NULL,
  `color_code` varchar(7) NOT NULL DEFAULT '#000000',
  `affects_balance` tinyint(1) NOT NULL DEFAULT '1',
  `is_active` tinyint(1) NOT NULL DEFAULT '1',
  `sort_order` int NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_vacation_types_name` (`name`),
  UNIQUE KEY `uq_vacation_types_system_name` (`system_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT INTO `vacation_types` (`id`, `name`, `system_name`, `color_code`, `affects_balance`, `is_active`, `sort_order`)
VALUES
  (uuid(), 'Основной оплачиваемый', 'paid', '#2f80ed', 1, 1, 1),
  (uuid(), 'За свой счёт', 'unpaid', '#eb5757', 0, 1, 2),
  (uuid(), 'Учебный', 'study', '#f2994a', 0, 1, 3);

ALTER TABLE `vacations`
  ADD COLUMN `vacation_type_id` varchar(36) DEFAULT NULL AFTER `status`,
  ADD KEY `idx_vacations_type` (`vacation_type_id`),
  ADD CONSTRAINT `fk_vacations_type` FOREIGN KEY (`vacation_type_id`) REFERENCES `vacation_types` (`id`) ON DELETE SET NULL ON UPDATE CASCADE;

-- Существующие отпуска по умолчанию считаем основным оплачиваемым типом.
UPDATE `vacations` v
JOIN `vacation_types` t ON t.system_name = 'paid'
SET v.vacation_type_id = t.id
WHERE v.vacation_type_id IS NULL;
