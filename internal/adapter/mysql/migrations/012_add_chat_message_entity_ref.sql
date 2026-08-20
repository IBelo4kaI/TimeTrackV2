-- Возможность сослаться на сущность (например, заявку на отпуск) прямо из
-- сообщения — в отличие от chats.entity_type/entity_id (весь чат привязан
-- к заявке, см. GetOrCreateEntityChat), тут ссылка на ОДНУ сущность внутри
-- ОДНОГО сообщения в обычном (не привязанном) чате: выбрал заявку, написал
-- текст — отправилось одним сообщением.
--
-- entity_title/entity_subtitle — снимок отображаемых полей на момент
-- отправки (например, "Заявка на отпуск" / "Иванов Иван · 01.08–15.08.2026 ·
-- На рассмотрении"), формирует фронт при выборе. Бэк ничего не знает о
-- вакациях/больничных — как и entity_type/entity_id, это просто непрозрачная
-- ссылка; собеседники видят то, чем поделился отправитель, даже если у них
-- самих нет прав читать сущность напрямую (переход по ссылке уже проверит
-- права как обычно, на странице самой заявки).
ALTER TABLE `chat_messages`
  ADD COLUMN `entity_type` varchar(63) DEFAULT NULL AFTER `body`,
  ADD COLUMN `entity_id` varchar(36) DEFAULT NULL AFTER `entity_type`,
  ADD COLUMN `entity_title` varchar(255) DEFAULT NULL AFTER `entity_id`,
  ADD COLUMN `entity_subtitle` varchar(255) DEFAULT NULL AFTER `entity_title`;
