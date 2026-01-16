package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaFS embed.FS

func Open(path string) (*sql.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("db path is required")
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err := applySchema(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func applySchema(ctx context.Context, db *sql.DB) error {
	schemaSQL, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}

	if _, err := db.ExecContext(ctx, string(schemaSQL)); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}

	if err := ensureParentTaskIDColumn(ctx, db); err != nil {
		return err
	}

	return nil
}

func ensureParentTaskIDColumn(ctx context.Context, db *sql.DB) error {
	var exists int
	err := db.QueryRowContext(ctx, "SELECT 1 FROM pragma_table_info('tasks') WHERE name = 'parent_task_id' LIMIT 1").Scan(&exists)
	if err == nil {
		if _, err := db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_tasks_parent_task_id ON tasks(parent_task_id)"); err != nil {
			return fmt.Errorf("create idx_tasks_parent_task_id: %w", err)
		}
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("check tasks.parent_task_id column: %w", err)
	}

	if _, err := db.ExecContext(ctx, "ALTER TABLE tasks ADD COLUMN parent_task_id INTEGER REFERENCES tasks(id) ON DELETE SET NULL"); err != nil {
		return fmt.Errorf("add tasks.parent_task_id column: %w", err)
	}

	if _, err := db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_tasks_parent_task_id ON tasks(parent_task_id)"); err != nil {
		return fmt.Errorf("create idx_tasks_parent_task_id: %w", err)
	}

	return nil
}
