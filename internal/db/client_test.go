package db_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"myai-novel-go/internal/db"
	"myai-novel-go/internal/testutil"
)

func TestMigrate_AddsWorkflowTaskScheduledAt(t *testing.T) {
	gdb := testutil.OpenTempDB(t)
	require.NoError(t, gdb.Migrator().DropTable("workflow_tasks"))
	require.NoError(t, gdb.Exec(`
		CREATE TABLE workflow_tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			book_id INTEGER NOT NULL,
			chapter_id INTEGER NOT NULL,
			chapter_no INTEGER NOT NULL,
			workflow_type TEXT NOT NULL,
			status TEXT NOT NULL,
			stage TEXT,
			lease_owner TEXT,
			lease_token TEXT,
			lease_expires_at TEXT,
			request_payload TEXT NOT NULL,
			result_payload TEXT,
			current_plan_id INTEGER,
			current_draft_id INTEGER,
			error_code TEXT,
			error_message TEXT,
			error_details TEXT,
			progress_percent INTEGER,
			attempt_count INTEGER NOT NULL DEFAULT 1,
			started_at TEXT,
			finished_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)
	`).Error)

	require.NoError(t, db.Migrate(gdb))
	cols, err := gdb.Migrator().ColumnTypes("workflow_tasks")
	require.NoError(t, err)
	found := false
	for _, c := range cols {
		if c.Name() == "scheduled_at" {
			found = true
			nullable, ok := c.Nullable()
			require.True(t, ok)
			require.False(t, nullable)
			break
		}
	}
	require.True(t, found)
}
