package mysql

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
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
		if !d.IsDir() && strings.HasSuffix(path, ".sql") {
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
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
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
