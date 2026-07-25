package payment

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"liki-web/internal/product"
)

// paymentProvider abstracts a payment gateway (Dodo, Xunhu, etc.).
type paymentProvider interface {
	CreateCheckout(ctx context.Context, product product.Product, amount int, orderID, email, returnURL string) (*CheckoutResult, error)
	VerifyWebhook(rawBody []byte, headers http.Header) (*WebhookEvent, error)
}

// Service handles payment provider operations: checkout creation and webhook verification.
// Domain-specific stores (naming/mingshu/hepan) handle their own order lifecycle.
type Service struct {
	dodo      paymentProvider
	xunhu     paymentProvider
	returnURL string
}

// NewService creates a payment service with the given dependencies.
func NewService(dodo, xunhu paymentProvider, returnURL string) *Service {
	return &Service{
		dodo:      dodo,
		xunhu:     xunhu,
		returnURL: returnURL,
	}
}

// Provider returns the payment provider for the given name.
func (s *Service) Provider(name string) (paymentProvider, error) {
	switch name {
	case ProviderDodo:
		return s.dodo, nil
	case ProviderXunhu:
		return s.xunhu, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownProvider, name)
	}
}

// ReturnURL returns the configured return URL for post-payment redirect.
func (s *Service) ReturnURL() string { return s.returnURL }

// VerifyWebhook attempts to verify a raw webhook payload against all registered providers.
// It returns the first matching event, or an error if all providers reject it.
func (s *Service) VerifyWebhook(body []byte, headers http.Header) (*WebhookEvent, error) {
	var errs []error
	for _, p := range []paymentProvider{s.dodo, s.xunhu} {
		event, err := p.VerifyWebhook(body, headers)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if event == nil {
			continue
		}
		return event, nil
	}
	return nil, fmt.Errorf("payment: verify webhook: %w", errors.Join(errs...))
}
