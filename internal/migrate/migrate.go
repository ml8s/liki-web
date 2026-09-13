// Package migrate provides a small forward-only SQL migration runner for the
// embedded SQLite databases. There is intentionally no down migration.
package migrate

import (
	"context"
	"database/sql"
	"fmt"
)

type Migration struct {
	ID string
	Up func(ctx context.Context, tx *sql.Tx) error
}

func Run(ctx context.Context, db *sql.DB, migrations []Migration) error {
	if _, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    scope TEXT NOT NULL,
    id TEXT NOT NULL,
    applied_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (scope, id)
)`); err != nil {
		return fmt.Errorf("migrate: create schema_migrations: %w", err)
	}
	for _, migration := range migrations {
		if migration.ID == "" || migration.Up == nil {
			return fmt.Errorf("migrate: invalid migration %+v", migration)
		}
		applied, err := isApplied(ctx, db, migration.ID)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if err := migration.Up(ctx, tx); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate: %s: %w", migration.ID, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(scope,id) VALUES ('liki',?)`, migration.ID); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate: record %s: %w", migration.ID, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migrate: commit %s: %w", migration.ID, err)
		}
	}
	return nil
}

func isApplied(ctx context.Context, db *sql.DB, id string) (bool, error) {
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_migrations WHERE scope='liki' AND id=?`, id).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}
