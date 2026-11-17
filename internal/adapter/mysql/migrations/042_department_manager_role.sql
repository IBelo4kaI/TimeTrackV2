ALTER TABLE department_managers ADD COLUMN auth_role_id VARCHAR(255) NOT NULL DEFAULT '';

INSERT INTO system_settings (setting_key, setting_value, setting_type, category, description, is_public)
VALUES ('department_manager_role_id', '', 'string', 'department_managers', 'ID роли руководителя отдела в сервисе авторизации', FALSE)
ON DUPLICATE KEY UPDATE setting_key = VALUES(setting_key);
