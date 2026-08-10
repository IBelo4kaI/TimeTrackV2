-- Системные (заданные в коде) категории файлов: у категории может быть
-- стабильный system_name, по которому бэкенд находит категорию для
-- автоматической раскладки файлов при загрузке.
-- Дерево: "Отпуска" (vacation) -> "Заявления" (application) — файлы отпуска
-- кладутся в подкатегорию; "Больничные" (medical) — файлы больничных кладутся
-- прямо в неё, без подкатегории.
-- Обычные категории, создаваемые админом руками, system_name не имеют (NULL).

ALTER TABLE `file_categories`
  ADD COLUMN `system_name` varchar(50) DEFAULT NULL AFTER `name`,
  ADD UNIQUE KEY `uq_file_categories_system_name` (`system_name`);

SET @vacation_category_id = UUID();
SET @application_category_id = UUID();
SET @medical_category_id = UUID();

INSERT INTO `file_categories` (`id`, `name`, `system_name`, `parent_id`, `color_code`, `sort_order`)
VALUES
  (@vacation_category_id, 'Отпуска', 'vacation', NULL, '#2f80ed', 1),
  (@application_category_id, 'Заявления', 'application', @vacation_category_id, '#2f80ed', 1),
  (@medical_category_id, 'Больничные', 'medical', NULL, '#eb5757', 2);

-- Бэкфилл: уже загруженные файлы отпусков и больничных раскладываем по
-- соответствующим категориям (только если категория ещё не проставлена).
UPDATE files f
JOIN file_entity_refs r ON r.file_id = f.id AND r.entity_type = 'vacation'
SET f.category_id = @application_category_id
WHERE f.category_id IS NULL;

UPDATE files f
JOIN file_entity_refs r ON r.file_id = f.id AND r.entity_type = 'sick_leave'
SET f.category_id = @medical_category_id
WHERE f.category_id IS NULL;
