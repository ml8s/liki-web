package sqliteutil

import (
	"path/filepath"
	"testing"
)

func TestOpenCreatesDirectoryAndAppliesPragmas(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "data")
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	for _, check := range []struct {
		name string
		want string
	}{
		{"journal_mode", "wal"},
		{"busy_timeout", "5000"},
		{"foreign_keys", "1"},
	} {
		var got string
		if err := db.QueryRow("PRAGMA " + check.name).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != check.want {
			t.Fatalf("%s = %q, want %q", check.name, got, check.want)
		}
	}
	if db.Stats().MaxOpenConnections != 1 {
		t.Fatalf("MaxOpenConnections = %d, want 1", db.Stats().MaxOpenConnections)
	}
}
