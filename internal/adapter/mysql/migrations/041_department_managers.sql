CREATE TABLE IF NOT EXISTS department_managers (
  id VARCHAR(36) NOT NULL PRIMARY KEY,
  company VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  department VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  user_id VARCHAR(36) NOT NULL,
  created_by VARCHAR(36) NOT NULL,
  role_sync_status ENUM('pending','not_configured','synced','failed') NOT NULL DEFAULT 'pending',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_department_manager (company, department, user_id),
  KEY idx_department_manager_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
