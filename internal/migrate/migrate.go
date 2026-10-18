// Package migrate provides a small forward-only SQL migration runner for the
// embedded SQLite databases. There is intentionally no down migration.
package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
)

var migrateMu sync.Mutex

type Migration struct {
	ID string
	Up func(ctx context.Context, tx *sql.Tx) error
}

// Run applies migrations in order, recording each in schema_migrations.
//
// ID uniqueness contract: the applied-key is (scope='liki', id), so every
// store sharing one database file must use globally unique IDs — a duplicate
// would make the later migration a silent no-op. New migrations carry a store
// prefix ("<store>_<NNN>_<slug>"); existing unprefixed IDs are frozen because
// renaming them would replay migrations on already-migrated databases. The
// shared-liki.db guard test (shared_liki_db_test.go) fails closed on
// collisions by asserting every store's artifacts exist.
func Run(ctx context.Context, db *sql.DB, migrations []Migration) error {
	migrateMu.Lock()
	defer migrateMu.Unlock()
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
		// The applied-check runs inside the transaction: under concurrent
		// migrators the loser fails on the (scope,id) primary key instead of
		// double-applying (fail-closed).
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		applied, err := isAppliedTx(ctx, tx, migration.ID)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		if applied {
			_ = tx.Rollback()
			continue
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

func isAppliedTx(ctx context.Context, tx *sql.Tx, id string) (bool, error) {
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_migrations WHERE scope='liki' AND id=?`, id).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}
