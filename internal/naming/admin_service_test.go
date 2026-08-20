package naming

import (
	"context"
	"testing"
)

func TestAdminService_ListOrders(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	oid := nmSeedOrder(t, s)
	s.MarkPaidIdempotent(context.Background(), oid, "pay-1") //nolint:errcheck

	svc := NewAdminService(s)
	orders, err := svc.ListOrders(context.Background(), "test@t.com")
	if err != nil {
		t.Fatalf("ListOrders: %v", err)
	}
	if len(orders) == 0 {
		t.Fatal("expected orders")
	}
	if orders[0].OrderID != oid {
		t.Errorf("order_id = %q", orders[0].OrderID)
	}
}

func TestAdminService_CreatePaidOrder(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	svc := NewAdminService(s)

	oid, err := svc.CreatePaidOrder(context.Background(), "buyer@t.com", "naming", 990, "CNY", "2099-12-31", "zh-Hans")
	if err != nil {
		t.Fatalf("CreatePaidOrder: %v", err)
	}
	if oid == "" {
		t.Fatal("empty order_id")
	}

	o, err := s.GetOrder(context.Background(), oid)
	if err != nil {
		t.Fatalf("GetOrder: %v", err)
	}
	if o.Status != "paid" {
		t.Errorf("status = %q", o.Status)
	}
}

func TestAdminService_UpdateExpire(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	oid := nmSeedOrder(t, s)
	svc := NewAdminService(s)

	if err := svc.UpdateExpire(context.Background(), oid, "2026-12-31"); err != nil {
		t.Fatalf("UpdateExpire: %v", err)
	}
	o, err := s.GetOrder(context.Background(), oid)
	if err != nil {
		t.Fatalf("GetOrder: %v", err)
	}
	if o.ChatExpiresAt != "2026-12-31" {
		t.Errorf("chat_expires_at = %q", o.ChatExpiresAt)
	}
}

func TestAdminService_GenerateVerificationCode(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	svc := NewAdminService(s)

	code, err := svc.GenerateVerificationCode(context.Background(), "admin")
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	if len(code) != 8 {
		t.Errorf("code length = %d", len(code))
	}
}

func TestAdminService_BatchGenerateVerificationCodes(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	svc := NewAdminService(s)

	codes, err := svc.BatchGenerateVerificationCodes(context.Background(), 3, "admin")
	if err != nil {
		t.Fatalf("BatchGenerate: %v", err)
	}
	if len(codes) != 3 {
		t.Errorf("expected 3 codes, got %d", len(codes))
	}
}

func TestAdminService_ListVerificationCodes(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	svc := NewAdminService(s)

	svc.GenerateVerificationCode(context.Background(), "admin") //nolint:errcheck
	svc.GenerateVerificationCode(context.Background(), "admin") //nolint:errcheck

	codes, err := svc.ListVerificationCodes(context.Background())
	if err != nil {
		t.Fatalf("ListCodes: %v", err)
	}
	if len(codes) < 2 {
		t.Errorf("expected >=2 codes, got %d", len(codes))
	}
}
