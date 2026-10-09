-- NULL: место не указано (отпуск, больничный и т.п.)
ALTER TABLE `user_time_entries` ADD COLUMN `work_location` VARCHAR(10) NULL DEFAULT NULL AFTER `hours_worked`;
