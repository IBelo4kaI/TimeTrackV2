ALTER TABLE vacations
ADD COLUMN manager_comment TEXT NULL,
ADD COLUMN status_updated_at TIMESTAMP NULL;

CREATE TABLE notifications (
  id VARCHAR(36) NOT NULL DEFAULT(uuid()),
  user_id VARCHAR(36) NOT NULL,
  title VARCHAR(255) NOT NULL,
  message TEXT NOT NULL,
  type ENUM('info', 'success', 'warn', 'error') DEFAULT 'info',
  is_read BOOLEAN DEFAULT FALSE,
  entity_type VARCHAR(50), -- 'vacation'
  entity_id VARCHAR(36), -- vacation.id
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_notifications_user (user_id, is_read)
);
