ALTER TABLE vacations
  MODIFY COLUMN status ENUM('pending','approved','rejected','withdrawn') NOT NULL DEFAULT 'pending';
ALTER TABLE vacations ADD COLUMN available_days INT NULL AFTER excess_days;
ALTER TABLE vacations ADD COLUMN decision_comment VARCHAR(4000) NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS vacation_history (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  vacation_id VARCHAR(36) NOT NULL,
  actor_id VARCHAR(36) NOT NULL,
  event_type VARCHAR(40) NOT NULL,
  detail TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_vacation_history (vacation_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO vacation_history (vacation_id, actor_id, event_type, detail, created_at)
SELECT id, '', 'created', 'Заявка создана (из архивной записи)', COALESCE(created_at, CURRENT_TIMESTAMP)
FROM vacations v
WHERE NOT EXISTS (SELECT 1 FROM vacation_history h WHERE BINARY h.vacation_id = BINARY v.id AND h.event_type = 'created');

INSERT INTO vacation_history (vacation_id, actor_id, event_type, detail, created_at)
SELECT r.entity_id, f.uploaded_by_user_id, 'file_added', CONCAT('Добавлен файл: ', f.original_name), f.created_at
FROM files f JOIN file_entity_refs r ON BINARY r.file_id = BINARY f.id
JOIN vacations v ON BINARY v.id = BINARY r.entity_id
WHERE r.entity_type = 'vacation' AND f.is_deleted = FALSE
  AND NOT EXISTS (
    SELECT 1 FROM vacation_history h
    WHERE BINARY h.vacation_id = BINARY r.entity_id AND h.event_type = 'file_added'
      AND BINARY h.detail = BINARY CONCAT('Добавлен файл: ', f.original_name)
      AND h.created_at = f.created_at
  );

INSERT INTO vacation_types (id, name, system_name, color_code, affects_balance, is_active, sort_order)
SELECT UUID(), 'Дополнительный', 'additional', '#8b5cf6', TRUE, TRUE, 4
WHERE NOT EXISTS (SELECT 1 FROM vacation_types WHERE system_name = 'additional' OR name = 'Дополнительный');
