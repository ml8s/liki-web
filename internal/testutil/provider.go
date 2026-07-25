package testutil

import (
	"context"
	"net/http"

	"liki-web/internal/payment"
	"liki-web/internal/product"
)

type MockPaymentProvider struct {
	CreateResult *payment.CheckoutResult
	CreateErr    error
	VerifyEvent  *payment.WebhookEvent
	VerifyErr    error
}

func (m *MockPaymentProvider) CreateCheckout(_ context.Context, _ product.Product, _ int, _, _, _ string) (*payment.CheckoutResult, error) {
	return m.CreateResult, m.CreateErr
}

func (m *MockPaymentProvider) VerifyWebhook(_ []byte, _ http.Header) (*payment.WebhookEvent, error) {
	return m.VerifyEvent, m.VerifyErr
}
