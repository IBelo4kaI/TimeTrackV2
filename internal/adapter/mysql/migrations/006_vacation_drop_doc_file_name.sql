-- Migrate existing vacation file references into the new files/file_entity_refs tables.
-- Files already on disk at docs/vacations/<doc_file_name> remain untouched.
-- We use a temporary column to generate UUIDs before inserting into `files`.
ALTER TABLE `vacations`
ADD COLUMN `_migrate_file_id` CHAR(36) NULL;

UPDATE `vacations`
SET
  `_migrate_file_id` = UUID()
WHERE
  `doc_file_name` IS NOT NULL
  AND `doc_file_name` != '';

INSERT INTO
  `files` (
    `id`,
    `original_name`,
    `storage_path`,
    `mime_type`,
    `file_type`,
    `size_bytes`,
    `checksum`,
    `uploaded_by_user_id`
  )
SELECT
  `_migrate_file_id`,
  `doc_file_name`,
  CONCAT('docs/vacations/', `doc_file_name`),
  'application/octet-stream',
  'document',
  0,
REPEAT ('0', 64),
`user_id`
FROM
  `vacations`
WHERE
  `_migrate_file_id` IS NOT NULL;

INSERT INTO
  `file_entity_refs` (`file_id`, `entity_type`, `entity_id`)
SELECT
  `_migrate_file_id`,
  'vacation',
  `id`
FROM
  `vacations`
WHERE
  `_migrate_file_id` IS NOT NULL;

ALTER TABLE `vacations`
DROP COLUMN `_migrate_file_id`;

ALTER TABLE `vacations`
DROP COLUMN `doc_file_name`;
