package mysql

import (
	"fmt"
	driver "github.com/go-sql-driver/mysql"
	"testing"
)

func TestResumeWorkflowAfterImplicitDDLCommit(t *testing.T) {
	duplicate := &driver.MySQLError{Number: 1060, Message: "Duplicate column name"}
	additions := []string{
		"ALTER TABLE vacations ADD COLUMN available_days INT NULL AFTER excess_days",
		"ALTER TABLE vacations ADD COLUMN decision_comment VARCHAR(4000) NOT NULL DEFAULT ''",
	}
	for _, stmt := range additions {
		if !resumableWorkflowColumn("migrations/039_vacation_workflow.sql", stmt, fmt.Errorf("wrapped: %w", duplicate)) {
			t.Fatalf("cannot resume %s", stmt)
		}
		if resumableWorkflowColumn("migrations/038_vacations_excess_days.sql", stmt, duplicate) {
			t.Fatal("must not ignore errors from other migrations")
		}
		if resumableWorkflowColumn("migrations/039_vacation_workflow.sql", stmt, &driver.MySQLError{Number: 1267}) {
			t.Fatal("must not hide collation errors")
		}
	}
	if resumableWorkflowColumn("migrations/039_vacation_workflow.sql", "ALTER TABLE vacations ADD COLUMN other INT", duplicate) {
		t.Fatal("must not ignore unrelated schema errors")
	}
}

func TestWorkflowMigrationStatements(t *testing.T) {
	raw, err := migrationsFS.ReadFile("migrations/039_vacation_workflow.sql")
	if err != nil {
		t.Fatal(err)
	}
	statements := splitStatements(string(raw))
	if len(statements) != 7 {
		t.Fatalf("got %d SQL statements, want 7", len(statements))
	}
}
