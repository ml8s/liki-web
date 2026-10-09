package feedback

import (
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

const samplePayload = `{"schema_version":"feedback-v1","meta":{"source":"skill-agent","skill":"liki","skill_version":"2026.09.14.0","engine_version":"2026.09.14.0"},"problem":{"type":"clarity","severity":"medium","summary":"resolved_period 解释入口不直观"}}`

func TestStore_CreateAndList(t *testing.T) {
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	s, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Create(&Feedback{Payload: samplePayload}) //nolint:errcheck
	if err != nil {
		t.Fatal(err)
	}

	list, err := s.List("", 10, 0) //nolint:errcheck
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("want 1, got %d", len(list))
	}
	if list[0].Payload != samplePayload {
		t.Fatalf("payload not preserved verbatim: %s", list[0].Payload)
	}
	if list[0].ID == "" || list[0].CreatedAt == "" {
		t.Fatalf("missing id or timestamp")
	}
	if list[0].Status != "pending" {
		t.Fatalf("default status should be pending, got %s", list[0].Status)
	}
}

func TestStore_DedupByHash(t *testing.T) {
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	s, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, ok := s.ExistingByHash("nope"); ok {
		t.Fatal("missing hash must not dedup")
	}
	if err := s.Create(&Feedback{Payload: samplePayload, Hash: sha256Hex([]byte(samplePayload))}); err != nil {
		t.Fatal(err)
	}
	id, status, ok := s.ExistingByHash(sha256Hex([]byte(samplePayload)))
	if !ok || id == "" || status != "pending" {
		t.Fatalf("expected dedup hit, got id=%q status=%q ok=%v", id, status, ok)
	}
}

func TestStore_ListByStatus(t *testing.T) {
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	s, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}

	s.Create(&Feedback{Payload: `{"problem":{"summary":"a"}}`}) //nolint:errcheck
	s.Create(&Feedback{Payload: `{"problem":{"summary":"b"}}`}) //nolint:errcheck
	s.Update("nonexistent", "done", "")                         //nolint:errcheck
	s.Create(&Feedback{Payload: `{"problem":{"summary":"c"}}`}) //nolint:errcheck

	pending, err := s.List("pending", 10, 0) //nolint:errcheck
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 3 {
		t.Fatalf("want 3 pending, got %d", len(pending))
	}
}

func TestStore_Update(t *testing.T) {
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	s, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}

	s.Create(&Feedback{Payload: `{"problem":{"summary":"test"}}`}) //nolint:errcheck
	list, _ := s.List("", 10, 0)                                   //nolint:errcheck
	id := list[0].ID

	err = s.Update(id, "done", "https://github.com/ml8s/liki/issues/1") //nolint:errcheck
	if err != nil {
		t.Fatal(err)
	}

	updated, _ := s.List("done", 10, 0) //nolint:errcheck
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
	defer func() { _ = db.Close() }()
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
	defer func() { _ = db.Close() }()
	s, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		s.Create(&Feedback{Payload: `{"problem":{"summary":"x"}}`}) //nolint:errcheck
	}
	list, err := s.List("", 3, 0) //nolint:errcheck
	if err != nil {
		t.Fatal(err)
	}
	if len(list) > 3 {
		t.Fatalf("want ≤3, got %d", len(list))
	}
}

func TestStore_ListOffsetPagination(t *testing.T) {
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	s, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if err := s.Create(&Feedback{Payload: `{"problem":{"summary":"x"}}`}); err != nil {
			t.Fatal(err)
		}
	}
	page1, err := s.List("", 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	page2, err := s.List("", 3, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 3 || len(page2) != 2 {
		t.Fatalf("want 3+2 pages, got %d+%d", len(page1), len(page2))
	}
	seen := map[string]bool{}
	for _, fb := range append(append([]Feedback{}, page1...), page2...) {
		if seen[fb.ID] {
			t.Fatalf("duplicate id across pages: %s", fb.ID)
		}
		seen[fb.ID] = true
	}
	if len(seen) != 5 {
		t.Fatalf("paged traversal must cover all rows, got %d", len(seen))
	}
	beyond, err := s.List("", 3, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(beyond) != 0 {
		t.Fatalf("offset past end must be empty, got %d", len(beyond))
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	f, err := os.CreateTemp("", "feedback-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	db, err := sql.Open("sqlite", f.Name()+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close(); _ = os.Remove(f.Name()) })
	return db
}
