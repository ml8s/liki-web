package main

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/ml8s/liki-web/internal/sqliteutil"
)

// appDatabases owns the bounded SQLite files opened by the process. Contexts
// own their schemas; this type only owns connection lifetime.
type appDatabases struct {
	dir      string
	Platform *sql.DB
	Profiles *sql.DB
	Feedback *sql.DB
	Liki     *sql.DB
	Aipay    *sql.DB
}

func openAppDatabases(dir string) (*appDatabases, error) {
	open := func(name string) (*sql.DB, error) {
		return sqliteutil.Open(filepath.Join(dir, name))
	}
	dbs := &appDatabases{dir: dir}
	var err error
	if dbs.Platform, err = open("platform.db"); err != nil {
		return nil, fmt.Errorf("open platform database: %w", err)
	}
	if dbs.Profiles, err = open("profiles.db"); err != nil {
		return nil, fmt.Errorf("open profiles database: %w", err)
	}
	if dbs.Feedback, err = open("feedback.db"); err != nil {
		return nil, fmt.Errorf("open feedback database: %w", err)
	}
	if dbs.Liki, err = open("liki.db"); err != nil {
		return nil, fmt.Errorf("open liki database: %w", err)
	}
	if dbs.Aipay, err = open("aipay.db"); err != nil {
		return nil, fmt.Errorf("open aipay database: %w", err)
	}
	return dbs, nil
}

func (d *appDatabases) Close() error {
	return errors.Join(
		d.Platform.Close(), d.Profiles.Close(), d.Feedback.Close(),
		d.Liki.Close(), d.Aipay.Close(),
	)
}
