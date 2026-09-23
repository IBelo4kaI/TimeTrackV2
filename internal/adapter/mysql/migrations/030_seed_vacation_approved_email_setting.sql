-- Адрес почты, на который слать письмо об утверждённой заявке на отпуск со
-- сканом заявления (см. vacationService.SendApprovalEmailIfReady). Настраивается
-- на странице "Настройки", как и notification_vacation_approved_user_ids —
-- только тут одно значение (email), а не список user_id.
INSERT INTO `system_settings`
  (`setting_key`, `setting_value`, `setting_type`, `category`, `description`)
VALUES
  ('notification_vacation_approved_email', '', 'string', 'notifications', 'Email, на который слать утверждённую заявку на отпуск со сканом (пусто — не слать)')
ON DUPLICATE KEY UPDATE setting_key = setting_key;
