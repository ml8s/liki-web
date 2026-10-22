package birthprofile

import (
	"context"
	"database/sql"
	"errors"
	"math"
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

func validInput() BirthProfileInput {
	lng := 121.4737
	tz := "Asia/Shanghai"
	return BirthProfileInput{
		Nickname: "妈妈",
		Birth: BirthFactsInput{
			Gender: "female", Date: "1958-05-12", Time: "06:00",
			Location: BirthLocationInput{Place: "上海", Longitude: &lng, Timezone: &tz},
		},
	}
}

func TestStoreCreateGetRoundtrip(t *testing.T) {
	store, userID := newStoreFixture(t)
	p, err := store.Create(context.Background(), userID, validInput())
	if err != nil {
		t.Fatal(err)
	}
	if p.Version != 1 || p.Nickname != "妈妈" || p.Birth.Gender != "female" {
		t.Fatalf("unexpected profile: %+v", p)
	}
	got, err := store.Get(context.Background(), userID, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != p.ID || got.Birth.Date != "1958-05-12" || got.Birth.Time != "06:00" {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
	if _, err := store.Get(context.Background(), "other_user", p.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-user read must 404, got %v", err)
	}
}

func TestStoreListCursorPagination(t *testing.T) {
	store, userID := newStoreFixture(t)
	for i := range 5 {
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
	for i := range MaxProfilesPerUser {
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
		mut  func(*BirthProfileInput)
	}{
		{"empty nickname", func(in *BirthProfileInput) { in.Nickname = "" }},
		{"bad gender", func(in *BirthProfileInput) { in.Birth.Gender = "other" }},
		{"bad date", func(in *BirthProfileInput) { in.Birth.Date = "58-05-12" }},
		{"future date", func(in *BirthProfileInput) { in.Birth.Date = "2999-01-01" }},
		{"missing time", func(in *BirthProfileInput) { in.Birth.Time = "" }},
		{"missing timezone", func(in *BirthProfileInput) { in.Birth.Location.Timezone = nil }},
		{"missing longitude", func(in *BirthProfileInput) { in.Birth.Location.Longitude = nil }},
		{"longitude out of range", func(in *BirthProfileInput) { lng := 181.0; in.Birth.Location.Longitude = &lng }},
		{"longitude NaN", func(in *BirthProfileInput) { lng := math.NaN(); in.Birth.Location.Longitude = &lng }},
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

func TestStore_AnonymizeUser(t *testing.T) {
	store, _ := newStoreFixture(t)

	// user_1 两个档案（含出生经度，擦除必须一并清空）
	for range 2 {
		in := validInput()
		lng := 121.4737
		in.Birth.Location.Longitude = &lng
		if _, err := store.Create(context.Background(), "user_1", in); err != nil {
			t.Fatal(err)
		}
	}
	// 同一库另一个用户一个档案（不受影响）
	other, err := store.Create(context.Background(), "user_2", validInput())
	if err != nil {
		t.Fatal(err)
	}

	n, err := store.AnonymizeUser(context.Background(), "user_1")
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("anonymize user_1: want 2, got %d", n)
	}
	var erasedLongitude sql.NullFloat64
	var erasedTimezone, erasedInstant sql.NullString
	if err := store.db.QueryRowContext(context.Background(),
		`SELECT longitude,timezone,instant FROM birth_profiles WHERE user_id='user_1' LIMIT 1`).Scan(&erasedLongitude, &erasedTimezone, &erasedInstant); err != nil {
		t.Fatal(err)
	}
	if erasedLongitude.Valid {
		t.Fatalf("anonymized profile must clear longitude, got %v", erasedLongitude.Float64)
	}
	if erasedTimezone.Valid || erasedInstant.Valid {
		t.Fatalf("anonymized profile must clear canonical fields, got tz=%v instant=%v", erasedTimezone, erasedInstant)
	}
	list, _, err := store.List(context.Background(), "user_1", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("user_1 profiles should be gone (deleted), got %d", len(list))
	}
	// 另一个用户的档案原样保留
	kept, err := store.Get(context.Background(), "user_2", other.ID)
	if err != nil {
		t.Fatal(err)
	}
	if kept.Nickname == "" || kept.Birth.Date == "" {
		t.Fatalf("user_2 profile must be untouched: %+v", kept)
	}
}

func TestStore_CleanupExpired(t *testing.T) {
	store, userID := newStoreFixture(t)
	p, err := store.Create(context.Background(), userID, validInput())
	if err != nil {
		t.Fatal(err)
	}
	// 把该档案置为 31 天前软删
	store.nowFunc = func() time.Time { return time.Now().UTC().Add(-31 * 24 * time.Hour) }
	if err := store.SoftDelete(context.Background(), userID, p.ID); err != nil {
		t.Fatal(err)
	}
	store.nowFunc = func() time.Time { return time.Now().UTC() }
	// 另一个 1 天前软删（30 天窗口内，不清）
	p2, err := store.Create(context.Background(), userID, validInput())
	if err != nil {
		t.Fatal(err)
	}
	store.nowFunc = func() time.Time { return time.Now().UTC().Add(-24 * time.Hour) }
	if err := store.SoftDelete(context.Background(), userID, p2.ID); err != nil {
		t.Fatal(err)
	}
	store.nowFunc = func() time.Time { return time.Now().UTC() }

	n, err := store.CleanupExpired(context.Background(), time.Now().UTC().Add(-30*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("cleanup: want 1 purged, got %d", n)
	}
}

func TestStoreBirthLocationRoundtrip(t *testing.T) {
	store, userID := newStoreFixture(t)
	in := validInput()
	lng := 121.4737
	in.Birth.Location.Longitude = &lng
	created, err := store.Create(context.Background(), userID, in)
	if err != nil {
		t.Fatal(err)
	}
	if created.Birth.Location.Longitude == nil || *created.Birth.Location.Longitude != lng {
		t.Fatalf("create longitude = %v, want %v", created.Birth.Location.Longitude, lng)
	}

	got, err := store.Get(context.Background(), userID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Birth.Location.Longitude == nil || *got.Birth.Location.Longitude != lng {
		t.Fatalf("get longitude = %v, want %v", got.Birth.Location.Longitude, lng)
	}

	list, _, err := store.List(context.Background(), userID, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Birth.Location.Longitude == nil || *list[0].Birth.Location.Longitude != lng {
		t.Fatalf("list longitude = %+v", list)
	}

	updated := validInput()
	updated.Birth.Location.Place = "深圳"
	updatedLng := 114.0579
	updated.Birth.Location.Longitude = &updatedLng
	saved, err := store.Update(context.Background(), userID, created.ID, created.Version, updated)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Birth.Location.Longitude == nil || *saved.Birth.Location.Longitude != updatedLng || saved.Birth.Location.Place != "深圳" {
		t.Fatalf("update = %+v", saved)
	}

}
