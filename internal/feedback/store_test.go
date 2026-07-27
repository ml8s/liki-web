package feedback

import (
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

func TestStore_CreateAndList(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	s, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Create(&Feedback{Category: "workflow", Message: "步骤4不清晰", Context: `{"step":4}`}) //nolint:errcheck
	if err != nil {
		t.Fatal(err)
	}

	list, err := s.List("", 10) //nolint:errcheck
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("want 1, got %d", len(list))
	}
	if list[0].Category != "workflow" || list[0].Message != "步骤4不清晰" {
		t.Fatalf("bad data: %+v", list[0])
	}
	if list[0].ID == "" || list[0].CreatedAt == "" {
		t.Fatalf("missing id or timestamp")
	}
	if list[0].Status != "pending" {
		t.Fatalf("default status should be pending, got %s", list[0].Status)
	}
}

func TestStore_ListByStatus(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	s, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}

	s.Create(&Feedback{Message: "a"}) //nolint:errcheck //nolint:errcheck
	s.Create(&Feedback{Message: "b"}) //nolint:errcheck
	s.Update("nonexistent", "done", "") //nolint:errcheck //nolint:errcheck
	s.Create(&Feedback{Message: "c"}) //nolint:errcheck

	pending, err := s.List("pending", 10) //nolint:errcheck
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 3 {
		t.Fatalf("want 3 pending, got %d", len(pending))
	}
}

func TestStore_Update(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	s, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}

	s.Create(&Feedback{Message: "test"}) //nolint:errcheck
	list, _ := s.List("", 10) //nolint:errcheck
	id := list[0].ID

	err = s.Update(id, "done", "https://github.com/ml8s/liki/issues/1") //nolint:errcheck
	if err != nil {
		t.Fatal(err)
	}

	updated, _ := s.List("done", 10) //nolint:errcheck
	if len(updated) != 1 {
		t.Fatalf("want 1 done, got %d", len(updated))
	}
	if updated[0].IssueURL != "https://github.com/ml8s/liki/issues/1" {
		t.Fatalf("bad issue_url: %s", updated[0].IssueURL)
	}

	err = s.Update(id, "done", "") //nolint:errcheck
	if err != nil {
		t.Fatal(err)
	}
}

func TestStore_UpdateNotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	s, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	err = s.Update("no-such-id", "done", "") //nolint:errcheck
	if err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestStore_ListLimit(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	s, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		s.Create(&Feedback{Message: "x"}) //nolint:errcheck
	}
	list, err := s.List("", 3) //nolint:errcheck
	if err != nil {
		t.Fatal(err)
	}
	if len(list) > 3 {
		t.Fatalf("want ≤3, got %d", len(list))
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	f, err := os.CreateTemp("", "feedback-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	db, err := sql.Open("sqlite", f.Name()+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close(); os.Remove(f.Name()) })
	return db
}
