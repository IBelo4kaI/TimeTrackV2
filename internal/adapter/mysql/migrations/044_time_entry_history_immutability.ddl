-- Existing matching triggers are verified and preserved by the migrator on retry.
-- Trigger DDL is separate because the sqlc MySQL parser does not support CREATE TRIGGER.
CREATE TRIGGER time_entry_history_no_update BEFORE UPDATE ON time_entry_history FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Time entry history is append-only';

CREATE TRIGGER time_entry_history_no_delete BEFORE DELETE ON time_entry_history FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Time entry history is append-only';
