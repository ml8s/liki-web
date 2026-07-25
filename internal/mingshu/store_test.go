package mingshu

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

func newLBTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewStore(db)
}

func seedLbOrder(t *testing.T, s *Store) string {
	t.Helper()
	oid := uuid.New().String()
	birth := "张三 | 1990-05-20 12:00 | 上海 | male"
	if err := s.CreateOrder(context.Background(), oid, "mingshu", 19900, "CNY", "test@t.com", birth, "zh-Hans"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return oid
}

func TestStore_CreateAndGetOrder(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	oid := seedLbOrder(t, s)

	o, err := s.GetOrder(context.Background(), oid)
	if err != nil {
		t.Fatalf("GetOrder: %v", err)
	}
	if o.Product != "mingshu" {
		t.Errorf("product = %q", o.Product)
	}
	if o.Amount != 19900 {
		t.Errorf("amount = %d", o.Amount)
	}
	if o.Status != "pending" {
		t.Errorf("status = %q", o.Status)
	}
	if o.Locale != "zh-Hans" {
		t.Errorf("locale = %q", o.Locale)
	}
	if !strings.Contains(o.Birth, "张三") {
		t.Errorf("birth = %q", o.Birth)
	}
}

func TestStore_GetOrder_NotFound(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	_, err := s.GetOrder(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStore_UpdateEmail(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	oid := seedLbOrder(t, s)

	s.UpdateEmail(context.Background(), oid, "new@t.com") //nolint:errcheck
	o, _ := s.GetOrder(context.Background(), oid) //nolint:errcheck
	if o.Email != "new@t.com" {
		t.Errorf("email = %q", o.Email)
	}
}

func TestStore_UpdateProvider(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	oid := seedLbOrder(t, s)

	s.UpdateProvider(context.Background(), oid, "xunhu") //nolint:errcheck
	o, _ := s.GetOrder(context.Background(), oid) //nolint:errcheck
	if o.Provider != "xunhu" {
		t.Errorf("provider = %q", o.Provider)
	}
}

func TestStore_MarkPaid(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	oid := seedLbOrder(t, s)

	paid, err := s.MarkPaidIdempotent(context.Background(), oid, "pay-1")
	if err != nil {
		t.Fatalf("MarkPaid: %v", err)
	}
	if !paid {
		t.Error("expected paid=true")
	}
	o, _ := s.GetOrder(context.Background(), oid) //nolint:errcheck
	if o.Status != "paid" {
		t.Errorf("status = %q", o.Status)
	}
}

func TestStore_MarkPaid_Idempotent(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	oid := seedLbOrder(t, s)

	s.MarkPaidIdempotent(context.Background(), oid, "pay-1") //nolint:errcheck
	paid, _ := s.MarkPaidIdempotent(context.Background(), oid, "pay-1") //nolint:errcheck
	if paid {
		t.Error("expected paid=false for duplicate")
	}
}

func TestStore_MarkPaidViaCallback(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	oid := seedLbOrder(t, s)

	newPayment, email, err := s.MarkPaidViaCallback(context.Background(), oid)
	if err != nil {
		t.Fatalf("MarkPaidViaCallback: %v", err)
	}
	if !newPayment {
		t.Error("expected newPayment=true")
	}
	if email != "test@t.com" {
		t.Errorf("email = %q", email)
	}
}

func TestStore_SaveAndGetBirth(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	oid := seedLbOrder(t, s)

	birth := "李四 | 1992-08-15 18:30 | 北京 | female"
	if err := s.SaveBirth(context.Background(), oid, birth); err != nil {
		t.Fatalf("SaveBirth: %v", err)
	}

	got, err := s.GetBirth(context.Background(), oid)
	if err != nil {
		t.Fatalf("GetBirth: %v", err)
	}
	if !strings.Contains(got, "李四") {
		t.Errorf("birth = %q", got)
	}
}

func TestStore_SaveAndGetReport(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	oid := seedLbOrder(t, s)

	report := `{"content":"测试报告内容"}`
	if err := s.SaveReport(context.Background(), oid, report); err != nil {
		t.Fatalf("SaveReport: %v", err)
	}

	got, err := s.GetReport(context.Background(), oid)
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}
	if !strings.Contains(got, "测试报告") {
		t.Errorf("report = %q", got)
	}
}

func TestStore_GetReport_NotFound(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	_, err := s.GetReport(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStore_SaveReport_Replace(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	oid := seedLbOrder(t, s)

	s.SaveReport(context.Background(), oid, `{"v":1}`) //nolint:errcheck
	s.SaveReport(context.Background(), oid, `{"v":2}`) //nolint:errcheck

	got, _ := s.GetReport(context.Background(), oid) //nolint:errcheck
	if !strings.Contains(got, `"v":2`) {
		t.Errorf("expected v2, got %s", got)
	}
}
