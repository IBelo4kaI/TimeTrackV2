-- Заводим строки заранее с правильным setting_type='json' (без этого
-- UpdateValueSystemSetting при первом сохранении создал бы их с дефолтным
-- 'string' — не критично для парсинга на бэке, но некорректно с точки
-- зрения самой настройки).
INSERT INTO `system_settings`
  (`setting_key`, `setting_value`, `setting_type`, `category`, `description`)
VALUES
  ('notification_vacation_admin_user_ids', '[]', 'json', 'notifications', 'user_id сотрудников, которым слать уведомления о новых заявках на отпуск'),
  ('notification_sick_leave_admin_user_ids', '[]', 'json', 'notifications', 'user_id сотрудников, которым слать уведомления о новых заявках на больничный')
ON DUPLICATE KEY UPDATE setting_key = setting_key;
