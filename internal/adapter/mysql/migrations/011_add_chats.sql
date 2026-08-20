-- Чаты: свободные личные/групповые (chats.entity_type/entity_id = NULL) и
-- обсуждения конкретной заявки (entity_type='vacation'/'sick_leave'/...,
-- entity_id=id заявки) — тот же полиморфный паттерн, что уже используется
-- в file_entity_refs для привязки файлов к заявкам.
--
-- Вложения к сообщениям отдельной таблицей не заводим — файл крепится к
-- сообщению через уже существующий file_entity_refs с
-- entity_type='chat_message', entity_id=<id сообщения>, точно так же, как
-- сейчас файлы крепятся к vacation/sick_leave.
--
-- chat_messages.id — bigint auto_increment, а не uuid() как везде в проекте:
-- сообщения высокочастотны и требуют естественно сортируемого id для
-- пагинации (WHERE id > :cursor) и курсора непрочитанного
-- (chat_participants.last_read_message_id); у UUID такой сортировки нет.
-- Сообщения неизменяемы (редактирования нет по требованиям, только
-- удаление) — поэтому updated_at им не нужен.

CREATE TABLE `chats` (
  `id` varchar(36) NOT NULL DEFAULT(uuid()),
  `type` enum('direct','group') NOT NULL,
  -- Название группового чата; для direct фронт сам берёт имя собеседника.
  `name` varchar(255) DEFAULT NULL,
  -- Привязка к заявке/сущности. NULL у обоих полей = свободный чат.
  `entity_type` varchar(63) DEFAULT NULL,
  `entity_id` varchar(36) DEFAULT NULL,
  `created_by_user_id` varchar(36) NOT NULL,
  -- Денормализация для сортировки списка чатов по активности без JOIN+MAX
  -- по chat_messages; обновляется сервисом при вставке нового сообщения.
  `last_message_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  -- Одна заявка — одно обсуждение. NULL у обоих полей не конфликтует
  -- (MySQL допускает много NULL в UNIQUE KEY), так что свободных чатов
  -- может быть сколько угодно.
  UNIQUE KEY `uq_chats_entity` (`entity_type`, `entity_id`),
  KEY `idx_chats_last_message` (`last_message_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `chat_messages` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `chat_id` varchar(36) NOT NULL,
  `sender_user_id` varchar(36) NOT NULL,
  `body` text NOT NULL,
  `is_deleted` tinyint(1) NOT NULL DEFAULT '0',
  `deleted_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_chat_messages_chat_id` (`chat_id`, `id`),
  CONSTRAINT `fk_chat_messages_chat` FOREIGN KEY (`chat_id`) REFERENCES `chats` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `chat_participants` (
  `chat_id` varchar(36) NOT NULL,
  `user_id` varchar(36) NOT NULL,
  -- 'admin' — управление чатом (переименовать группу, добавить/убрать
  -- участников), не путать с правами системы (vacation.all и т.п.).
  `role` enum('member','admin') NOT NULL DEFAULT 'member',
  -- Курсор "прочитано до сообщения №X" — дешевле, чем строка на каждое
  -- сообщение x каждого читателя. Непрочитанные считаются как
  -- COUNT(*) FROM chat_messages WHERE chat_id=? AND id > last_read_message_id.
  `last_read_message_id` bigint unsigned DEFAULT NULL,
  `last_read_at` timestamp NULL DEFAULT NULL,
  `joined_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`chat_id`, `user_id`),
  KEY `idx_chat_participants_user` (`user_id`),
  CONSTRAINT `fk_chat_participants_chat` FOREIGN KEY (`chat_id`) REFERENCES `chats` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_chat_participants_last_read` FOREIGN KEY (`last_read_message_id`) REFERENCES `chat_messages` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
