// Package sqliteutil contains the standard SQLite connection policy used by
// every Liki database file.
package sqliteutil

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Open opens path with the settings required by all Liki databases: WAL for
// concurrent readers, a busy timeout for startup races, FK enforcement inside
// each bounded-context database, and one connection because SQLite permits a
// single writer at a time.
func Open(path string) (*sql.DB, error) {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("sqliteutil: create data directory: %w", err)
		}
	}

	db, err := sql.Open(
		"sqlite",
		path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)",
	)
	if err != nil {
		return nil, fmt.Errorf("sqliteutil: open %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	return db, nil
}
