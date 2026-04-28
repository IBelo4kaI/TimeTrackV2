--
-- Структура таблицы `calendar_events`
--
CREATE TABLE `calendar_events` (
  `id` varchar(36) NOT NULL DEFAULT(uuid()),
  `event_date` date NOT NULL,
  `day_type_id` varchar(36) NOT NULL,
  `description` varchar(255) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- --------------------------------------------------------
--
-- Структура таблицы `day_types`
--
CREATE TABLE `day_types` (
  `id` varchar(36) NOT NULL DEFAULT(uuid()),
  `name` varchar(50) NOT NULL,
  `system_name` varchar(50) NOT NULL,
  `is_work_day` tinyint(1) NOT NULL DEFAULT '1',
  `affects_vacation` tinyint(1) NOT NULL DEFAULT '1',
  `is_user_select` tinyint(1) NOT NULL DEFAULT '1',
  `color_code` varchar(7) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '#000000',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- --------------------------------------------------------
--
-- Структура таблицы `system_settings`
--
CREATE TABLE `system_settings` (
  `id` int NOT NULL,
  `setting_key` varchar(100) NOT NULL,
  `setting_value` text,
  `setting_type` enum('string', 'integer', 'boolean', 'float', 'json') DEFAULT 'string',
  `category` varchar(50) DEFAULT 'general',
  `description` varchar(255) DEFAULT NULL,
  `is_public` tinyint(1) DEFAULT '0',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- --------------------------------------------------------
--
-- Структура таблицы `user_time_entries`
--
CREATE TABLE `user_time_entries` (
  `id` varchar(36) NOT NULL DEFAULT(uuid()),
  `user_id` varchar(36) NOT NULL,
  `entry_date` date NOT NULL,
  `day_type_id` varchar(36) NOT NULL,
  `hours_worked` decimal(5, 2) NOT NULL DEFAULT '0.00',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- --------------------------------------------------------
--
-- Структура таблицы `vacations`
--
CREATE TABLE `vacations` (
  `id` varchar(36) NOT NULL DEFAULT(uuid()),
  `user_id` varchar(36) NOT NULL,
  `start_date` date NOT NULL,
  `end_date` date NOT NULL,
  `total_days` int NOT NULL,
  `description` text,
  `doc_file_name` text,
  `status` enum('pending', 'approved', 'rejected') DEFAULT 'pending',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- --------------------------------------------------------
--
-- Структура таблицы `sick_leaves`
--
CREATE TABLE `sick_leaves` (
  `id` varchar(36) NOT NULL DEFAULT(uuid()),
  `user_id` varchar(36) NOT NULL,
  `start_date` date NOT NULL,
  `end_date` date NOT NULL,
  `total_days` int NOT NULL,
  `description` text,
  `doc_file_name` text,
  `status` enum('pending', 'approved', 'rejected') DEFAULT 'pending',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- --------------------------------------------------------
--
-- Структура таблицы `work_standards`
--
CREATE TABLE `work_standards` (
  `id` varchar(36) NOT NULL DEFAULT(uuid()),
  `user_id` varchar(36) DEFAULT NULL,
  `month` int NOT NULL,
  `year` int NOT NULL,
  `standard_hours` int NOT NULL DEFAULT '0',
  `standard_days` int NOT NULL DEFAULT '0',
  `gender` int NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

--
-- Индексы сохранённых таблиц
--
--
-- Индексы таблицы `calendar_events`
--
ALTER TABLE `calendar_events`
ADD PRIMARY KEY (`id`),
ADD KEY `idx_calendar_events_date` (`event_date`),
ADD KEY `calendar_events_ibfk_1` (`day_type_id`);

--
-- Индексы таблицы `day_types`
--
ALTER TABLE `day_types`
ADD PRIMARY KEY (`id`),
ADD UNIQUE KEY `name` (`name`),
ADD UNIQUE KEY `system_name` (`system_name`);

--
-- Индексы таблицы `system_settings`
--
ALTER TABLE `system_settings`
ADD PRIMARY KEY (`id`),
ADD UNIQUE KEY `setting_key` (`setting_key`);

--
-- Индексы таблицы `user_time_entries`
--
ALTER TABLE `user_time_entries`
ADD PRIMARY KEY (`id`),
ADD UNIQUE KEY `unique_user_date` (`user_id`, `entry_date`),
ADD KEY `idx_user_time_entries_user_date` (`user_id`, `entry_date`),
ADD KEY `idx_user_time_entries_date` (`entry_date`),
ADD KEY `user_time_entries_ibfk_1` (`day_type_id`);

--
-- Индексы таблицы `vacations`
--
ALTER TABLE `vacations`
ADD PRIMARY KEY (`id`),
ADD KEY `idx_vacations_user_status` (`user_id`, `status`),
ADD KEY `idx_vacations_dates` (`start_date`, `end_date`);

--
-- Индексы таблицы `sick_leaves`
--
ALTER TABLE `sick_leaves`
ADD PRIMARY KEY (`id`),
ADD KEY `idx_sick_leaves_user_status` (`user_id`, `status`),
ADD KEY `idx_sick_leaves_dates` (`start_date`, `end_date`);

--
-- Индексы таблицы `work_standards`
--
ALTER TABLE `work_standards`
ADD PRIMARY KEY (`id`),
ADD UNIQUE KEY `unique_standard` (`month`, `year`, `gender`, `user_id`),
ADD KEY `idx_work_standards_period` (`year`, `month`);

--
-- AUTO_INCREMENT для сохранённых таблиц
--
--
-- AUTO_INCREMENT для таблицы `system_settings`
--
ALTER TABLE `system_settings`
MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- Ограничения внешнего ключа сохраненных таблиц
--
--
-- Ограничения внешнего ключа таблицы `calendar_events`
--
ALTER TABLE `calendar_events`
ADD CONSTRAINT `calendar_events_ibfk_1` FOREIGN KEY (`day_type_id`) REFERENCES `day_types` (`id`) ON DELETE CASCADE ON UPDATE CASCADE;

--
-- Ограничения внешнего ключа таблицы `user_time_entries`
--
ALTER TABLE `user_time_entries`
ADD CONSTRAINT `user_time_entries_ibfk_1` FOREIGN KEY (`day_type_id`) REFERENCES `day_types` (`id`) ON DELETE CASCADE ON UPDATE CASCADE;
