CREATE TABLE IF NOT EXISTS department_manager_merged_assignments (
  id VARCHAR(36) NOT NULL PRIMARY KEY,
  company VARCHAR(255) NOT NULL,
  department VARCHAR(255) NOT NULL,
  user_id VARCHAR(36) NOT NULL,
  created_by VARCHAR(36) NOT NULL,
  role_sync_status ENUM('pending','not_configured','synced','failed') NOT NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  auth_role_id VARCHAR(255) NOT NULL,
  retained_id VARCHAR(36) NOT NULL,
  role_released BOOLEAN NOT NULL DEFAULT FALSE,
  KEY idx_department_manager_merged_retained (retained_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

INSERT INTO department_manager_merged_assignments
  (id, company, department, user_id, created_by, role_sync_status, created_at, updated_at, auth_role_id, retained_id)
SELECT m.id, m.company, m.department, m.user_id, m.created_by, m.role_sync_status,
       m.created_at, m.updated_at, m.auth_role_id, duplicates.retained_id
FROM department_managers m
JOIN (SELECT department, user_id, MIN(id) AS retained_id
      FROM department_managers GROUP BY department, user_id HAVING COUNT(*) > 1) duplicates
  ON duplicates.department = m.department AND duplicates.user_id = m.user_id
LEFT JOIN department_manager_merged_assignments archived ON archived.id = m.id
WHERE m.id <> duplicates.retained_id AND archived.id IS NULL;

UPDATE department_managers retained
JOIN department_manager_merged_assignments archived ON archived.retained_id = retained.id
SET retained.role_sync_status = 'pending', retained.updated_at = retained.updated_at
WHERE archived.role_released = FALSE AND archived.auth_role_id <> ''
  AND archived.auth_role_id <> retained.auth_role_id;

DELETE m FROM department_managers m
JOIN department_manager_merged_assignments archived ON archived.id = m.id
JOIN department_managers retained ON retained.id = archived.retained_id
WHERE m.department = retained.department AND m.user_id = retained.user_id;

ALTER TABLE department_managers
  DROP INDEX uq_department_manager,
  ADD UNIQUE KEY uq_department_manager (department, user_id);
