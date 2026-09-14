package profile

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func newStoreFixture(t *testing.T) (*Store, string) {
	t.Helper()
	store := NewStore(openTestDB(t))
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return store, "user_1"
}

func validInput() Input {
	return Input{
		Nickname: "妈妈", Gender: "female", Calendar: "lunar",
		BirthDate: "1958-闰5-12", BirthTime: "06:00", BirthTimeKnown: "known",
		BirthPlace: "上海",
	}
}

func TestStoreCreateGetRoundtrip(t *testing.T) {
	store, userID := newStoreFixture(t)
	p, err := store.Create(context.Background(), userID, validInput())
	if err != nil {
		t.Fatal(err)
	}
	if p.Version != 1 || p.Nickname != "妈妈" || p.Gender != "female" {
		t.Fatalf("unexpected profile: %+v", p)
	}
	got, err := store.Get(context.Background(), userID, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != p.ID || got.BirthDate != "1958-闰5-12" {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
	if _, err := store.Get(context.Background(), "other_user", p.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-user read must 404, got %v", err)
	}
}

func TestStoreListCursorPagination(t *testing.T) {
	store, userID := newStoreFixture(t)
	for i := 0; i < 5; i++ {
		in := validInput()
		in.Nickname = string(rune('A' + i))
		if _, err := store.Create(context.Background(), userID, in); err != nil {
			t.Fatal(err)
		}
	}
	page1, next, err := store.List(context.Background(), userID, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 2 || next == "" {
		t.Fatalf("page1 len=%d next=%q", len(page1), next)
	}
	page2, next2, err := store.List(context.Background(), userID, next, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 2 || next2 == "" {
		t.Fatalf("page2 len=%d next2=%q", len(page2), next2)
	}
	page3, next3, err := store.List(context.Background(), userID, next2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page3) != 1 || next3 != "" {
		t.Fatalf("page3 len=%d next3=%q", len(page3), next3)
	}
	seen := map[string]bool{}
	for _, p := range append(append(page1, page2...), page3...) {
		if seen[p.ID] {
			t.Fatalf("duplicate id across pages: %s", p.ID)
		}
		seen[p.ID] = true
	}
}

func TestStoreUpdateOptimisticLock(t *testing.T) {
	store, userID := newStoreFixture(t)
	p, err := store.Create(context.Background(), userID, validInput())
	if err != nil {
		t.Fatal(err)
	}
	in := validInput()
	in.Nickname = "新称呼"
	if _, err := store.Update(context.Background(), userID, p.ID, 999, in); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale version must conflict, got %v", err)
	}
	updated, err := store.Update(context.Background(), userID, p.ID, p.Version, in)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 2 || updated.Nickname != "新称呼" {
		t.Fatalf("update mismatch: %+v", updated)
	}
}

func TestStoreSoftDelete(t *testing.T) {
	store, userID := newStoreFixture(t)
	p, err := store.Create(context.Background(), userID, validInput())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SoftDelete(context.Background(), userID, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(context.Background(), userID, p.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted profile must 404, got %v", err)
	}
	if err := store.SoftDelete(context.Background(), userID, p.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("double delete must 404, got %v", err)
	}
}

func TestStoreQuota(t *testing.T) {
	store, userID := newStoreFixture(t)
	for i := 0; i < MaxProfilesPerUser; i++ {
		if _, err := store.Create(context.Background(), userID, validInput()); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}
	if _, err := store.Create(context.Background(), userID, validInput()); !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("over quota must fail, got %v", err)
	}
}

func TestInputValidation(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*Input)
	}{
		{"empty nickname", func(in *Input) { in.Nickname = "" }},
		{"bad gender", func(in *Input) { in.Gender = "other" }},
		{"bad calendar", func(in *Input) { in.Calendar = "julian" }},
		{"bad known", func(in *Input) { in.BirthTimeKnown = "maybe" }},
		{"bad solar date", func(in *Input) { in.Calendar = "solar"; in.BirthDate = "58-05-12" }},
		{"future date", func(in *Input) { in.Calendar = "solar"; in.BirthDate = "2999-01-01" }},
		{"missing time", func(in *Input) { in.BirthTime = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validInput()
			tc.mut(&in)
			if err := in.Validate(nowForTest()); err == nil {
				t.Fatalf("expected validation error for %s", tc.name)
			}
		})
	}
}

func nowForTest() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}
