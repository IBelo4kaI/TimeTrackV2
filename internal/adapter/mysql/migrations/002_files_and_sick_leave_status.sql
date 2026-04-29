CREATE TABLE IF NOT EXISTS `files` (
  `id` CHAR(36) NOT NULL,
  `original_name` VARCHAR(255) NOT NULL,
  `storage_path` VARCHAR(500) NOT NULL,
  `mime_type` VARCHAR(127) NOT NULL,
  `file_type` VARCHAR(63) NOT NULL,
  `size_bytes` BIGINT NOT NULL,
  `checksum` CHAR(64) NOT NULL,
  `uploaded_by_user_id` CHAR(36) NOT NULL,
  `is_deleted` TINYINT(1) NOT NULL DEFAULT '0',
  `deleted_at` TIMESTAMP NULL DEFAULT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_files_checksum` (`checksum`),
  KEY `idx_files_uploaded_by` (`uploaded_by_user_id`),
  KEY `idx_files_is_deleted` (`is_deleted`),
  KEY `idx_files_file_type` (`file_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `file_entity_refs` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `file_id` CHAR(36) NOT NULL,
  `entity_type` VARCHAR(63) NOT NULL,
  `entity_id` VARCHAR(36) NOT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_file_entity_refs` (`file_id`,`entity_type`,`entity_id`),
  KEY `idx_file_entity_refs_entity` (`entity_type`,`entity_id`),
  CONSTRAINT `fk_file_entity_refs_file` FOREIGN KEY (`file_id`) REFERENCES `files` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

UPDATE `sick_leaves` SET `status` = 'unofficial' WHERE `status` IN ('pending', 'rejected');

UPDATE `sick_leaves` SET `status` = 'official' WHERE `status` = 'approved';

ALTER TABLE `sick_leaves`
  MODIFY COLUMN `status` ENUM('official','unofficial') NOT NULL DEFAULT 'unofficial';
