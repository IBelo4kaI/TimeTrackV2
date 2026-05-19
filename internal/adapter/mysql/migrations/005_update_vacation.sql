ALTER TABLE vacations
  MODIFY COLUMN `status` enum('pending','approved','rejected') NOT NULL DEFAULT 'pending';
