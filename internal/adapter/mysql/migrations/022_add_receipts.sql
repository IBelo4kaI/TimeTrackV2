-- Чеки, полученные через "Проверка чека онлайн" (ФНС) API. Сотрудник
-- сканирует QR на кассовом чеке на фронте, фронт сам ходит во внешнее API и
-- присылает нам уже готовый разобранный ответ — мы только сохраняем
-- (см. internal/receipt).
--
-- DATETIME (не TIMESTAMP) — та же причина, что и в
-- 018_notification_timestamp_utc.sql/021_news.sql: TIMESTAMP пересчитывается
-- через сессионную time_zone. Для created_at/updated_at это уже чинили,
-- здесь дополнительно и ticket_date — это дата покупки из чека, а не время
-- на сервере, пересчитывать её через сессионный time_zone нельзя.
--
-- Суммы — в копейках (BIGINT), как отдаёт API, чтобы не ловить ошибки
-- округления float; на фронте делить на 100.
--
-- INT, а не TINYINT(1), для operation_type/taxation_type — в проекте
-- TINYINT(1) маппится sqlc в bool, а это коды (1-4 и 1/2/4/8/16/32
-- соответственно), не флаги. INT — как и везде в проекте для похожих
-- кодов/счётчиков (total_days, sort_order и т.п.), чтобы sqlc стабильно
-- сгенерировал int32.

CREATE TABLE `receipts` (
  `id` VARCHAR(36) NOT NULL DEFAULT (uuid()),
  `user_id` VARCHAR(36) NOT NULL,

  -- Фискальные реквизиты — уникальный "ключ" чека, ФНС отдаёт втроём.
  `fiscal_drive_number` VARCHAR(20) NOT NULL,
  `fiscal_document_number` VARCHAR(20) NOT NULL,
  `fiscal_sign` VARCHAR(20) NOT NULL,

  `ticket_date` DATETIME NOT NULL,
  `total_sum` BIGINT NOT NULL,
  `seller_inn` VARCHAR(12) NOT NULL,
  `seller_name` VARCHAR(255) DEFAULT NULL,
  `operation_type` INT NOT NULL,

  `retail_place_address` VARCHAR(500) DEFAULT NULL,
  `request_number` VARCHAR(20) DEFAULT NULL,
  `cash_total_sum` BIGINT DEFAULT NULL,
  `ecash_total_sum` BIGINT DEFAULT NULL,
  `taxation_type` INT DEFAULT NULL,

  `nds20` BIGINT NOT NULL DEFAULT '0',
  `nds10` BIGINT NOT NULL DEFAULT '0',
  `nds0` BIGINT NOT NULL DEFAULT '0',
  `nds_no` BIGINT NOT NULL DEFAULT '0',

  -- исходная qr-строка, если фронт её передал — для отладки.
  `raw_qr` TEXT,

  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL,

  PRIMARY KEY (`id`),
  -- защита от повторной загрузки одного и того же чека
  UNIQUE KEY `uq_receipts_fiscal` (`fiscal_drive_number`, `fiscal_document_number`, `fiscal_sign`),
  KEY `idx_receipts_user` (`user_id`),
  KEY `idx_receipts_ticket_date` (`ticket_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- id — bigint auto_increment, а не uuid() как у receipts: позиции чисто
-- служебные (нет своего API-эндпоинта по id), тот же подход, что и у
-- chat_messages (см. комментарий в 011_add_chats.sql).
CREATE TABLE `receipt_items` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `receipt_id` VARCHAR(36) NOT NULL,

  `position_no` INT DEFAULT NULL,
  `name` VARCHAR(500) NOT NULL,
  `price` BIGINT NOT NULL,
  `quantity` DECIMAL(10,3) NOT NULL,
  `sum` BIGINT NOT NULL,

  PRIMARY KEY (`id`),
  KEY `idx_receipt_items_receipt` (`receipt_id`),
  CONSTRAINT `fk_receipt_items_receipt` FOREIGN KEY (`receipt_id`) REFERENCES `receipts` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
