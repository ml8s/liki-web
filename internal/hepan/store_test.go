package hepan

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"liki-web/internal/httputil"
	"liki-web/internal/product"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	for _, stmt := range strings.Split(Schema, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("exec schema: %v", err)
		}
	}
	return NewStore(db)
}

func seedOrder(t *testing.T, store *Store) string {
	t.Helper()
	orderID := product.NewOrderID()
	pA, _ := json.Marshal(PersonInfo{Name: "张三", Birth: "1990-05-20", City: "上海", Gender: "male"}) //nolint:errcheck
	pB, _ := json.Marshal(PersonInfo{Name: "李四", Birth: "1992-08-15", City: "北京", Gender: "female"}) //nolint:errcheck
	if err := store.CreateOrder(context.Background(), orderID, "hepan", 990, "CNY", "test@t.com", string(pA), string(pB), "zh-Hans"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return orderID
}

func TestStore_CreateAndGetOrder(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	orderID := seedOrder(t, store)

	order, err := store.GetOrder(context.Background(), orderID)
	if err != nil {
		t.Fatalf("GetOrder: %v", err)
	}
	if order.Product != "hepan" {
		t.Errorf("product = %q", order.Product)
	}
	if order.Amount != 990 {
		t.Errorf("amount = %d", order.Amount)
	}
	if order.Currency != "CNY" {
		t.Errorf("currency = %q", order.Currency)
	}
	if order.Status != "pending" {
		t.Errorf("status = %q", order.Status)
	}
	if order.Locale != "zh-Hans" {
		t.Errorf("locale = %q", order.Locale)
	}

	var a, b PersonInfo
	json.Unmarshal([]byte(order.PersonA), &a) //nolint:errcheck
	json.Unmarshal([]byte(order.PersonB), &b) //nolint:errcheck
	if a.Name != "张三" || b.Name != "李四" {
		t.Errorf("person_a=%+v person_b=%+v", a, b)
	}
}

func TestStore_GetOrder_NotFound(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, err := store.GetOrder(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	if order != nil {
		t.Fatal("expected nil order")
	}
	appErr, ok := err.(*httputil.AppError)
	if !ok {
		t.Fatal("expected *AppError")
	}
	if appErr.HTTPStatus != 404 {
		t.Errorf("status = %d, want 404", appErr.HTTPStatus)
	}
}

func TestStore_UpdateEmail(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	orderID := seedOrder(t, store)

	if err := store.UpdateEmail(context.Background(), orderID, "new@t.com"); err != nil {
		t.Fatalf("UpdateEmail: %v", err)
	}

	order, _ := store.GetOrder(context.Background(), orderID) //nolint:errcheck
	if order.Email != "new@t.com" {
		t.Errorf("email = %q", order.Email)
	}
}

func TestStore_UpdateProvider(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	orderID := seedOrder(t, store)

	if err := store.UpdateProvider(context.Background(), orderID, "xunhu"); err != nil {
		t.Fatalf("UpdateProvider: %v", err)
	}

	order, _ := store.GetOrder(context.Background(), orderID) //nolint:errcheck
	if order.Provider != "xunhu" {
		t.Errorf("provider = %q", order.Provider)
	}
}

func TestStore_MarkPaid(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	orderID := seedOrder(t, store)

	// First call → marks paid
	newPayment, email, err := store.MarkPaidViaCallback(context.Background(), orderID)
	if err != nil {
		t.Fatalf("MarkPaid first: %v", err)
	}
	if !newPayment {
		t.Error("expected newPayment=true")
	}
	_ = email

	order, _ := store.GetOrder(context.Background(), orderID) //nolint:errcheck
	if order.Status != "paid" {
		t.Errorf("status = %q, want paid", order.Status)
	}
}

func TestStore_MarkPaid_Idempotent(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	orderID := seedOrder(t, store)

	store.MarkPaidViaCallback(context.Background(), orderID) //nolint:errcheck

	// Second call → no-op
	newPayment, _, _ := store.MarkPaidViaCallback(context.Background(), orderID) //nolint:errcheck
	if newPayment {
		t.Error("expected newPayment=false on duplicate")
	}
}

func TestStore_SaveAndGetReport(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	orderID := seedOrder(t, store)

	report := `{"content":"测试报告","order_id":"` + orderID + `"}`
	if err := store.SaveReport(context.Background(), orderID, report); err != nil {
		t.Fatalf("SaveReport: %v", err)
	}

	got, err := store.GetReport(context.Background(), orderID)
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}
	if !strings.Contains(got, "测试报告") {
		t.Errorf("report = %q", got)
	}
}

func TestStore_GetReport_NotFound(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	_, err := store.GetReport(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStore_SaveReport_Replace(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	orderID := seedOrder(t, store)

	store.SaveReport(context.Background(), orderID, `{"v":1}`) //nolint:errcheck
	store.SaveReport(context.Background(), orderID, `{"v":2}`) //nolint:errcheck

	got, _ := store.GetReport(context.Background(), orderID) //nolint:errcheck
	if !strings.Contains(got, `"v":2`) {
		t.Errorf("expected v2, got %s", got)
	}
}

func TestStore_CreateOrder_AmountCurrency(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	pA, _ := json.Marshal(PersonInfo{Name: "A"}) //nolint:errcheck
	pB, _ := json.Marshal(PersonInfo{Name: "B"}) //nolint:errcheck

	tests := []struct {
		amount   int
		currency string
	}{
		{990, "CNY"},
		{990, "USD"},
	}
	for _, tc := range tests {
		oid := product.NewOrderID()
		if err := store.CreateOrder(context.Background(), oid, "hepan", tc.amount, tc.currency, "e@t.com", string(pA), string(pB), "en"); err != nil {
			t.Errorf("CreateOrder(%s, %d): %v", tc.currency, tc.amount, err)
		}
		o, _ := store.GetOrder(context.Background(), oid) //nolint:errcheck //nolint:errcheck
		if o.Amount != tc.amount {
			t.Errorf("amount = %d, want %d", o.Amount, tc.amount)
		}
	}
}

func TestStore_MarkPaidIdempotent(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	oid := seedOrder(t, s)

	paid, err := s.MarkPaidIdempotent(context.Background(), oid, "pay-1")
	if err != nil {
		t.Fatalf("MarkPaidIdempotent: %v", err)
	}
	if !paid {
		t.Error("expected paid=true")
	}
	o, _ := s.GetOrder(context.Background(), oid) //nolint:errcheck
	if o.Status != "paid" {
		t.Errorf("status = %q", o.Status)
	}

	// Second call with same paymentID → idempotent
	paid, err = s.MarkPaidIdempotent(context.Background(), oid, "pay-1")
	if err != nil {
		t.Fatalf("MarkPaidIdempotent duplicate: %v", err)
	}
	if paid {
		t.Error("expected paid=false for duplicate")
	}
}
