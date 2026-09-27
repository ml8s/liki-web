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
	// Force the file (and WAL/SHM sidecars) to exist before tightening
	// permissions, because sql.Open is lazy and creates files on first use.
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqliteutil: ping %s: %w", path, err)
	}
	if err := tightenPermissions(path); err != nil {
		_ = db.Close()
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

// tightenPermissions restricts the database file (and its WAL/SHM sidecars,
// if present) to the owning user. SQLite may create files before this call
// completes, so this is best-effort per open: it covers newly created files
// and converges permissions on existing ones.
func tightenPermissions(path string) error {
	for _, candidate := range []string{path, path + "-wal", path + "-shm"} {
		info, err := os.Stat(candidate)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("sqliteutil: stat %s: %w", candidate, err)
		}
		// Only tighten; never widen. Preserve execute bits for a directory,
		// though sqliteutil.Open is documented for files.
		mode := info.Mode().Perm() & 0o700
		if err := os.Chmod(candidate, mode); err != nil {
			return fmt.Errorf("sqliteutil: chmod %s: %w", candidate, err)
		}
	}
	return nil
}
