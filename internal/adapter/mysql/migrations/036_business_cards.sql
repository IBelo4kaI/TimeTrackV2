CREATE TABLE `business_cards` (
  `id` VARCHAR(36) NOT NULL DEFAULT (uuid()),
  -- полный номер, AES-GCM (см. internal/business_card/crypto.go)
  `card_number_enc` VARCHAR(512) NOT NULL,
  `last4` VARCHAR(4) NOT NULL,
  `label` VARCHAR(255) DEFAULT NULL,
  `bank` VARCHAR(255) DEFAULT NULL,
  `holder_name` VARCHAR(255) DEFAULT NULL,
  `expiry` VARCHAR(10) DEFAULT NULL,
  -- в копейках
  `card_limit` BIGINT DEFAULT NULL,
  -- active / blocked
  `status` VARCHAR(20) NOT NULL DEFAULT 'active',
  `created_at` DATETIME(6) NOT NULL,
  `updated_at` DATETIME(6) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- released_at IS NULL — текущий владелец
CREATE TABLE `business_card_assignments` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `card_id` VARCHAR(36) NOT NULL,
  `user_id` VARCHAR(36) NOT NULL,
  `assigned_by` VARCHAR(36) NOT NULL,
  `assigned_at` DATETIME(6) NOT NULL,
  `released_at` DATETIME(6) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_bca_card` (`card_id`),
  KEY `idx_bca_user` (`user_id`),
  CONSTRAINT `fk_bca_card` FOREIGN KEY (`card_id`) REFERENCES `business_cards` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE `receipts`
  ADD COLUMN `business_card_id` VARCHAR(36) DEFAULT NULL,
  ADD KEY `idx_receipts_business_card` (`business_card_id`),
  ADD CONSTRAINT `fk_receipts_business_card` FOREIGN KEY (`business_card_id`) REFERENCES `business_cards` (`id`) ON DELETE SET NULL;
