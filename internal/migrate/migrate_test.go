package migrate_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/ml8s/liki-web/internal/migrate"
)

func TestRunIsIdempotent(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	calls := 0
	migrations := []migrate.Migration{{
		ID: "001_test",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			calls++
			_, err := tx.ExecContext(ctx, `CREATE TABLE test_rows (id TEXT PRIMARY KEY)`)
			return err
		},
	}}
	for range 2 {
		if err := migrate.Run(ctx, db, migrations); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}
