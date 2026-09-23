-- Отметка "письмо с утверждённой заявкой и сканом уже отправлено" — чтобы
-- не слать повторно, если после отправки к заявке добавят ещё файл (см.
-- vacationService.SendApprovalEmailIfReady).
ALTER TABLE `vacations`
  ADD COLUMN `approval_email_sent_at` TIMESTAMP NULL DEFAULT NULL AFTER `status`;
