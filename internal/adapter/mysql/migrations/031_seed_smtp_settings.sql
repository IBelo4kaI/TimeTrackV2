-- Настройки SMTP для отправки писем (см. internal/smtp_settings,
-- internal/mail) — переехали из env в "Настройки", как и остальные.
-- smtp_password хранится зашифрованным (AES-GCM, ключ — SMTP_ENCRYPTION_KEY
-- из env), в API/UI никогда не отдаётся обратно в открытом виде.
INSERT INTO `system_settings`
  (`setting_key`, `setting_value`, `setting_type`, `category`, `description`)
VALUES
  ('smtp_host', '', 'string', 'smtp', 'Адрес SMTP-сервера для отправки писем'),
  ('smtp_port', '587', 'integer', 'smtp', 'Порт SMTP-сервера'),
  ('smtp_username', '', 'string', 'smtp', 'Логин для авторизации на SMTP-сервере'),
  ('smtp_password', '', 'string', 'smtp', 'Пароль для авторизации на SMTP-сервере (хранится зашифрованным)'),
  ('smtp_from', '', 'string', 'smtp', 'Адрес отправителя (заголовок From)')
ON DUPLICATE KEY UPDATE setting_key = setting_key;
