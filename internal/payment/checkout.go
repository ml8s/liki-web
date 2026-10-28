package payment

import (
	"context"
	"errors"
	"fmt"

	"github.com/ml8s/liki-web/internal/metrics"
	"time"

	"github.com/google/uuid"

	"github.com/ml8s/liki-web/internal/httputil"
)

func (r *Runtime) CreateIntent(ctx context.Context, input CreateIntentInput) (PaymentIntent, error) {
	flow, ok := r.flows[input.Product]
	if !ok {
		return PaymentIntent{}, fmt.Errorf("%w: %s", ErrUnknownProduct, input.Product)
	}
	order, err := flow.PrepareCheckout(ctx, input.BusinessOrderID)
	if err != nil {
		return PaymentIntent{}, err
	}
	if order.AmountMinor <= 0 {
		return PaymentIntent{}, fmt.Errorf("payment: business order %s has invalid amount", input.BusinessOrderID)
	}
	if order.Email == "" {
		return PaymentIntent{}, fmt.Errorf("payment: business order %s has no email", input.BusinessOrderID)
	}
	if order.Currency == "" {
		return PaymentIntent{}, fmt.Errorf("payment: business order %s has no currency", input.BusinessOrderID)
	}
	if order.UserID == "" && input.UserID != "" {
		order.UserID = input.UserID
	}
	provider, err := r.routePolicy.Select(order.Currency, r.availableProviders())
	if err != nil {
		return PaymentIntent{}, err
	}

	orderLocale := order.Locale
	if orderLocale == "" {
		orderLocale = input.Locale
	}
	if orderLocale == "" {
		orderLocale = "zh-Hant"
	}

	intent := PaymentIntent{
		ID:              uuid.NewString(),
		BusinessOrderID: input.BusinessOrderID,
		UserID:          order.UserID,
		Product:         input.Product,
		Provider:        provider,
		Status:          IntentCreated,
		AmountMinor:     order.AmountMinor,
		Currency:        order.Currency,
		Email:           order.Email,
		Locale:          orderLocale,
		Metadata: map[string]string{
			MetadataBusinessOrderID: input.BusinessOrderID,
			MetadataProduct:         string(input.Product),
		},
	}
	intent.SuccessURL = flow.SuccessRedirect(intent)
	intent.FailureURL = flow.FailureRedirect(intent)
	if err := r.intents.CreateIntent(ctx, intent); err != nil {
		return PaymentIntent{}, err
	}
	metrics.Default.Inc("liki_web_payment_intent_created_total", map[string]string{"provider": string(intent.Provider)})
	return intent, nil
}

func (r *Runtime) StartCheckout(ctx context.Context, intentID string) (*CheckoutResult, PaymentIntent, error) {
	intent, err := r.intents.GetIntent(ctx, intentID)
	if err != nil {
		return nil, PaymentIntent{}, err
	}
	if intent.Status == IntentPaid {
		return nil, intent, errors.New("payment: intent already paid")
	}
	gateway, err := r.gateway(intent.Provider)
	if err != nil {
		return nil, intent, err
	}
	flow, ok := r.flows[intent.Product]
	if !ok {
		return nil, intent, fmt.Errorf("%w: %s", ErrUnknownProduct, intent.Product)
	}

	req := CheckoutRequest{
		IntentID:        intent.ID,
		BusinessOrderID: intent.BusinessOrderID,
		Product:         intent.Product,
		AmountMinor:     intent.AmountMinor,
		Currency:        intent.Currency,
		Email:           intent.Email,
		Locale:          intent.Locale,
		Title:           "Liki",
		Metadata: map[string]string{
			MetadataPaymentIntentID: intent.ID,
			MetadataBusinessOrderID: intent.BusinessOrderID,
			MetadataProduct:         string(intent.Product),
		},
		ReturnURL: absoluteURL(r.baseURL, "/api/payments/return/"+intent.ID),
		CancelURL: absoluteURL(r.baseURL, flow.FailureRedirect(intent)),
	}
	if err := req.validateURLs(); err != nil {
		return nil, intent, err
	}
	result, err := gateway.CreateCheckout(ctx, req)
	if err != nil {
		return nil, intent, err
	}
	updated, err := r.intents.MarkCheckoutCreated(ctx, intent.ID, *result)
	if err != nil {
		return nil, intent, err
	}
	token, tokenErr := httputil.NewCapabilityToken("payment-status", intent.ID, time.Now(), 24*time.Hour)
	if tokenErr != nil {
		return nil, updated, tokenErr
	}
	result.StatusToken = token
	return result, updated, nil
}

func (r *Runtime) GetIntent(ctx context.Context, id string) (PaymentIntent, error) {
	return r.intents.GetIntent(ctx, id)
}

// GetIntents resolves several intents in one query for read-side annotations.
func (r *Runtime) GetIntents(ctx context.Context, ids []string) (map[string]PaymentIntent, error) {
	return r.intents.GetIntents(ctx, ids)
}
