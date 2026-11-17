ALTER TABLE vacations
  ADD COLUMN department_status ENUM('not_required','pending','approved','rejected') NOT NULL DEFAULT 'not_required',
  ADD COLUMN department_actor_id VARCHAR(36) NOT NULL DEFAULT '',
  ADD COLUMN department_decided_at TIMESTAMP NULL DEFAULT NULL,
  ADD COLUMN department_comment VARCHAR(4000) NOT NULL DEFAULT '';
