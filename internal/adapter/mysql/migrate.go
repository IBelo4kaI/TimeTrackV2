package mysql

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"io/fs"
	"sort"
	"strings"
)

//go:embed migrations/*.sql migrations/*.ddl
var migrationsFS embed.FS

// RunMigrations applies all pending SQL files from migrations/ in lexicographic
// order. Applied files are tracked in the schema_migrations table.
func RunMigrations(ctx context.Context, db *sql.DB) error {
	if err := ensureMigrationsTable(ctx, db); err != nil {
		return fmt.Errorf("ensure migrations table: %w", err)
	}

	files, err := collectSQLFiles()
	if err != nil {
		return err
	}

	applied, err := loadApplied(ctx, db)
	if err != nil {
		return err
	}

	for _, name := range files {
		if applied[name] {
			continue
		}
		if err := applyFile(ctx, db, name); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
		fmt.Printf("migration applied: %s\n", name)
	}
	return nil
}

func ensureMigrationsTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name       VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (name)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	return err
}

func collectSQLFiles() ([]string, error) {
	var names []string
	err := fs.WalkDir(migrationsFS, "migrations", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && (strings.HasSuffix(path, ".sql") || strings.HasSuffix(path, ".ddl")) {
			names = append(names, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk migrations dir: %w", err)
	}
	sort.Strings(names)
	return names, nil
}

func loadApplied(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT name FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("load applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}
	return applied, rows.Err()
}

func applyFile(ctx context.Context, db *sql.DB, path string) error {
	data, err := migrationsFS.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	for _, stmt := range splitStatements(string(data)) {
		if path == "migrations/043_timesheet_approval_audit.sql" {
			handled, err := ensureTimeEntryForeignKey(ctx, tx, stmt)
			if err != nil {
				return fmt.Errorf("ensure audited time entry FK: %w", err)
			}
			if handled {
				continue
			}
		}
		if path == "migrations/044_time_entry_history_immutability.ddl" {
			if err := ensureTimeEntryHistoryTrigger(ctx, tx, stmt); err != nil {
				return fmt.Errorf("ensure audit trigger: %w", err)
			}
			continue
		}
		if path == "migrations/045_department_manager_department_only.sql" && strings.HasPrefix(stmt, "ALTER TABLE department_managers") {
			if err := ensureDepartmentOnlyManagerIndex(ctx, tx, stmt); err != nil {
				return fmt.Errorf("ensure department manager uniqueness: %w", err)
			}
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			if resumableWorkflowColumn(path, stmt, err) {
				continue
			}
			return fmt.Errorf("exec statement: %w\nSQL: %s", err, stmt)
		}
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES (?)`, path); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	return tx.Commit()
}

// splitStatements splits SQL content by semicolons, skipping comments and empty lines.
func splitStatements(content string) []string {
	var stmts []string
	var current strings.Builder

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		current.WriteString(line)
		current.WriteByte('\n')
		if strings.HasSuffix(trimmed, ";") {
			stmt := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(current.String()), ";"))
			if stmt != "" {
				stmts = append(stmts, stmt)
			}
			current.Reset()
		}
	}

	if stmt := strings.TrimSpace(current.String()); stmt != "" {
		stmts = append(stmts, stmt)
	}

	return stmts
}

// MySQL commits ALTER/CREATE independently of the enclosing transaction.
// Only the listed single-column additions can safely resume after partial DDL.
func resumableWorkflowColumn(path, stmt string, err error) bool {
	if path != "migrations/039_vacation_workflow.sql" && path != "migrations/042_department_manager_role.sql" {
		return false
	}
	var dbErr *mysql.MySQLError
	if !errors.As(err, &dbErr) || dbErr.Number != 1060 {
		return false
	}
	if path == "migrations/042_department_manager_role.sql" {
		return stmt == "ALTER TABLE department_managers ADD COLUMN auth_role_id VARCHAR(255) NOT NULL DEFAULT ''"
	}
	return stmt == "ALTER TABLE vacations ADD COLUMN available_days INT NULL AFTER excess_days" ||
		stmt == "ALTER TABLE vacations ADD COLUMN decision_comment VARCHAR(4000) NOT NULL DEFAULT ''"
}

// Migration 044 never drops an existing audit guard while resuming partial DDL.
func ensureTimeEntryHistoryTrigger(ctx context.Context, tx *sql.Tx, stmt string) error {
	const action = "SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Time entry history is append-only'"
	var name, event string
	for _, operation := range []string{"UPDATE", "DELETE"} {
		triggerName := "time_entry_history_no_" + strings.ToLower(operation)
		expected := "CREATE TRIGGER " + triggerName + " BEFORE " + operation + " ON time_entry_history FOR EACH ROW " + action
		if stmt == expected {
			name, event = triggerName, operation
			break
		}
	}
	if name == "" {
		return errors.New("unexpected statement in audit trigger migration")
	}
	verify := func() (bool, error) {
		var timing, operation, table, body string
		err := tx.QueryRowContext(ctx, `SELECT ACTION_TIMING, EVENT_MANIPULATION, EVENT_OBJECT_TABLE, ACTION_STATEMENT
FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA = DATABASE() AND TRIGGER_NAME = ?`, name).Scan(&timing, &operation, &table, &body)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if timing != "BEFORE" || operation != event || table != "time_entry_history" || !strings.EqualFold(strings.Join(strings.Fields(body), " "), action) {
			return false, fmt.Errorf("audit trigger %s has an unexpected definition", name)
		}
		return true, nil
	}
	exists, err := verify()
	if err != nil || exists {
		return err
	}
	_, err = tx.ExecContext(ctx, stmt)
	if err != nil {
		var dbErr *mysql.MySQLError
		if errors.As(err, &dbErr) && dbErr.Number == 1359 {
			exists, verifyErr := verify()
			if verifyErr != nil {
				return verifyErr
			}
			if exists {
				return nil
			}
		}
		return err
	}
	return nil
}

func ensureTimeEntryForeignKey(ctx context.Context, tx *sql.Tx, stmt string) (bool, error) {
	const replacement = "user_time_entries_day_type_audit_fk"
	const old = "user_time_entries_ibfk_1"
	const add = "ALTER TABLE user_time_entries ADD CONSTRAINT " + replacement + " FOREIGN KEY (day_type_id) REFERENCES day_types (id) ON DELETE RESTRICT ON UPDATE RESTRICT"
	const drop = "ALTER TABLE user_time_entries DROP FOREIGN KEY " + old
	normalized := strings.Join(strings.Fields(stmt), " ")
	if normalized != add && normalized != drop {
		return false, nil
	}
	verify := func(name string, restrict bool) (bool, error) {
		var onDelete, onUpdate, column, parent, parentColumn string
		var count int
		err := tx.QueryRowContext(ctx, `SELECT rc.DELETE_RULE, rc.UPDATE_RULE, COUNT(*),
MAX(k.COLUMN_NAME), MAX(k.REFERENCED_TABLE_NAME), MAX(k.REFERENCED_COLUMN_NAME)
FROM information_schema.REFERENTIAL_CONSTRAINTS rc
JOIN information_schema.KEY_COLUMN_USAGE k
ON k.CONSTRAINT_SCHEMA = rc.CONSTRAINT_SCHEMA AND k.CONSTRAINT_NAME = rc.CONSTRAINT_NAME AND k.TABLE_NAME = rc.TABLE_NAME
WHERE rc.CONSTRAINT_SCHEMA = DATABASE() AND rc.TABLE_NAME = 'user_time_entries' AND rc.CONSTRAINT_NAME = ?
AND k.REFERENCED_TABLE_SCHEMA = DATABASE()
GROUP BY rc.DELETE_RULE, rc.UPDATE_RULE`, name).Scan(&onDelete, &onUpdate, &count, &column, &parent, &parentColumn)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if count != 1 || column != "day_type_id" || parent != "day_types" || parentColumn != "id" || (restrict && (onDelete != "RESTRICT" || onUpdate != "RESTRICT")) {
			return false, fmt.Errorf("foreign key %s has an unexpected definition", name)
		}
		return true, nil
	}
	exists, err := verify(replacement, true)
	if err != nil {
		return true, err
	}
	if normalized == add {
		if exists {
			return true, nil
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			exists, verifyErr := verify(replacement, true)
			if verifyErr != nil {
				return true, verifyErr
			}
			if exists {
				return true, nil
			}
			return true, err
		}
		return true, nil
	}
	if !exists {
		return true, errors.New("replacement foreign key is missing; refusing to drop the old constraint")
	}
	exists, err = verify(old, false)
	if err != nil || !exists {
		return true, err
	}
	if _, err := tx.ExecContext(ctx, stmt); err != nil {
		exists, verifyErr := verify(old, false)
		if verifyErr != nil {
			return true, verifyErr
		}
		if !exists {
			return true, nil
		}
		return true, err
	}
	return true, nil
}

func ensureDepartmentOnlyManagerIndex(ctx context.Context, tx *sql.Tx, stmt string) error {
	var columns string
	var nonUnique int
	err := tx.QueryRowContext(ctx, `SELECT GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX), MAX(NON_UNIQUE)
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'department_managers' AND INDEX_NAME = 'uq_department_manager'`).Scan(&columns, &nonUnique)
	if err != nil {
		return err
	}
	if nonUnique != 0 {
		return errors.New("department manager index is not unique")
	}
	if columns == "department,user_id" {
		return nil
	}
	if columns != "company,department,user_id" {
		return fmt.Errorf("unexpected department manager index: %s", columns)
	}
	_, err = tx.ExecContext(ctx, stmt)
	return err
}
