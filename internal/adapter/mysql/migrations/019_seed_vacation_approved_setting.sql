-- Получатели уведомления "заявка на отпуск утверждена" — отдельный список
-- от notification_vacation_admin_user_ids (тот про НОВЫЕ заявки, этот про
-- решение по уже поданной): например, бухгалтерия, которой не нужно
-- участвовать в рассмотрении заявок, но нужно знать об утверждённых.
INSERT INTO `system_settings`
  (`setting_key`, `setting_value`, `setting_type`, `category`, `description`)
VALUES
  ('notification_vacation_approved_user_ids', '[]', 'json', 'notifications', 'user_id сотрудников, которым слать уведомление об утверждении заявки на отпуск (с ФИО заявителя и датами)')
ON DUPLICATE KEY UPDATE setting_key = setting_key;
