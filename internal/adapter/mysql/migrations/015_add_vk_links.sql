-- Связь сотрудника с его VK-аккаунтом — куда дублировать уведомления
-- (см. internal/vk). Один сотрудник — один VK-аккаунт и наоборот.
CREATE TABLE `user_vk_links` (
  `user_id` varchar(36) NOT NULL,
  `vk_user_id` bigint NOT NULL,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`user_id`),
  UNIQUE KEY `uq_user_vk_links_vk_user_id` (`vk_user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
