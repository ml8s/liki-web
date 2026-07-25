package payment

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"liki-web/internal/product"
)

type mockPaymentProvider struct {
	createResult *CheckoutResult
	createErr    error
	verifyEvent  *WebhookEvent
	verifyErr    error
}

func (m *mockPaymentProvider) CreateCheckout(_ context.Context, _ product.Product, _ int, _, _, _ string) (*CheckoutResult, error) {
	return m.createResult, m.createErr
}

func (m *mockPaymentProvider) VerifyWebhook(_ []byte, _ http.Header) (*WebhookEvent, error) {
	return m.verifyEvent, m.verifyErr
}

func newTestSvc(t *testing.T) (*Service, *mockPaymentProvider, *mockPaymentProvider) {
	t.Helper()
	dodoMock := &mockPaymentProvider{
		createResult: &CheckoutResult{CheckoutURL: "https://pay.example.com/checkout"},
	}
	xunhuMock := &mockPaymentProvider{
		createResult: &CheckoutResult{CheckoutURL: "https://pay.xunhu.com/checkout", QRCodeURL: "https://pay.xunhu.com/qr"},
	}
	svc := NewService(dodoMock, xunhuMock, "https://liki.hk")
	return svc, dodoMock, xunhuMock
}

// ── NewService ──

func TestNewService_ReturnsService(t *testing.T) {
	t.Parallel()
	svc, _, _ := newTestSvc(t)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

// ── Provider ──

func TestProvider_Dodo(t *testing.T) {
	t.Parallel()
	svc, _, _ := newTestSvc(t)
	p, err := svc.Provider(ProviderDodo)
	if err != nil {
		t.Fatalf("Provider(dodo): %v", err)
	}
	if p == nil {
		t.Error("expected non-nil provider")
	}
}

func TestProvider_Xunhu(t *testing.T) {
	t.Parallel()
	svc, _, _ := newTestSvc(t)
	p, err := svc.Provider(ProviderXunhu)
	if err != nil {
		t.Fatalf("Provider(xunhu): %v", err)
	}
	if p == nil {
		t.Error("expected non-nil provider")
	}
}

func TestProvider_Unknown(t *testing.T) {
	t.Parallel()
	svc, _, _ := newTestSvc(t)
	_, err := svc.Provider("unknown")
	if !errors.Is(err, ErrUnknownProvider) {
		t.Errorf("expected ErrUnknownProvider, got %v", err)
	}
}

// ── VerifyWebhook ──

func TestVerifyWebhook_ReturnsEvent(t *testing.T) {
	t.Parallel()
	svc, dodoMock, xunhuMock := newTestSvc(t)
	dodoMock.verifyErr = errors.New("bad signature")
	xunhuMock.verifyEvent = &WebhookEvent{
		Type: EventPaymentSucceeded,
		Data: WebhookEventData{OrderID: "order-1", PaymentID: "pay-1"},
	}

	event, err := svc.VerifyWebhook([]byte(`{}`), http.Header{})
	if err != nil {
		t.Fatalf("VerifyWebhook: %v", err)
	}
	if event.Type != EventPaymentSucceeded {
		t.Errorf("type = %q", event.Type)
	}
}

func TestVerifyWebhook_AllProvidersReject(t *testing.T) {
	t.Parallel()
	svc, dodoMock, xunhuMock := newTestSvc(t)
	dodoMock.verifyErr = errors.New("bad signature")
	xunhuMock.verifyErr = errors.New("bad hash")

	_, err := svc.VerifyWebhook([]byte(`{}`), http.Header{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ── ReturnURL ──

func TestReturnURL_ReturnsConfiguredURL(t *testing.T) {
	t.Parallel()
	svc, _, _ := newTestSvc(t)
	if url := svc.ReturnURL(); url != "https://liki.hk" {
		t.Errorf("ReturnURL = %q", url)
	}
}
