CREATE TABLE IF NOT EXISTS timesheet_months (
  user_id VARCHAR(36) NOT NULL,
  year SMALLINT NOT NULL,
  month TINYINT NOT NULL,
  status ENUM('pending','approved','needs_review') NOT NULL DEFAULT 'pending',
  revision BIGINT NOT NULL DEFAULT 0,
  approved_revision BIGINT NULL,
  approved_by VARCHAR(36) NULL,
  approved_at TIMESTAMP NULL DEFAULT NULL,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, year, month),
  CONSTRAINT chk_timesheet_month CHECK (month BETWEEN 1 AND 12 AND year BETWEEN 1 AND 9999),
  CONSTRAINT chk_timesheet_revision CHECK (revision >= 0 AND (approved_revision IS NULL OR approved_revision <= revision)),
  CONSTRAINT chk_timesheet_approved CHECK (status <> 'approved' OR (approved_revision IS NOT NULL AND approved_revision = revision AND approved_by IS NOT NULL AND approved_by <> '' AND approved_at IS NOT NULL))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS time_entry_history (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id VARCHAR(36) NOT NULL,
  entry_date DATE NOT NULL,
  entry_id VARCHAR(36) NULL,
  actor_id VARCHAR(36) NULL,
  actor_type ENUM('employee','manager','admin','system') NOT NULL,
  source VARCHAR(80) NOT NULL,
  operation ENUM('create','update','delete') NOT NULL,
  before_values JSON NULL,
  after_values JSON NULL,
  comment VARCHAR(4000) NOT NULL DEFAULT '',
  month_revision BIGINT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_time_entry_history_month (user_id, entry_date, id),
  INDEX idx_time_entry_history_entry (entry_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Add the replacement first; the migrator verifies existing constraints on retry.
ALTER TABLE user_time_entries
  ADD CONSTRAINT user_time_entries_day_type_audit_fk FOREIGN KEY (day_type_id) REFERENCES day_types (id) ON DELETE RESTRICT ON UPDATE RESTRICT;

ALTER TABLE user_time_entries DROP FOREIGN KEY user_time_entries_ibfk_1;
