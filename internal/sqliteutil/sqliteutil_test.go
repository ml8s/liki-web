package sqliteutil

import (
	"os"
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

	for _, sidecar := range []string{"test.db", "test.db-wal", "test.db-shm"} {
		info, err := os.Stat(filepath.Join(dir, sidecar))
		if err != nil && os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm&0o077 != 0 {
			t.Fatalf("%s mode = %04o, want group/other bits clear", sidecar, perm)
		}
	}
}
