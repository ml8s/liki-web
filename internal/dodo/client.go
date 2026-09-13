package dodo

import (
	"context"
	"fmt"
	"net/http"

	dodopayments "github.com/dodopayments/dodopayments-go"
	"github.com/dodopayments/dodopayments-go/option"

	"liki-web/internal/payment"
	"liki-web/internal/product"
)

// Client wraps the Dodo Payments SDK for checkout and webhook operations.
type Client struct {
	checkoutSvc *dodopayments.CheckoutSessionService
	webhookSvc  *dodopayments.WebhookService
	paymentsSvc *dodopayments.PaymentService
	products    map[product.Product]string
}

// New creates a Dodo Payments client with the given API key and webhook secret.
func New(apiKey, webhookKey string, testMode bool, products map[product.Product]string) *Client {
	opts := []option.RequestOption{option.WithBearerToken(apiKey)}
	if testMode {
		opts = append(opts, option.WithEnvironmentTestMode())
	} else {
		opts = append(opts, option.WithEnvironmentLiveMode())
	}
	return &Client{
		checkoutSvc: dodopayments.NewCheckoutSessionService(opts...),
		webhookSvc:  dodopayments.NewWebhookService(append(opts, option.WithWebhookKey(webhookKey))...),
		paymentsSvc: dodopayments.NewPaymentService(opts...),
		products:    products,
	}
}

// CreateCheckout creates a Dodo checkout session from the normalized request.
func (c *Client) CreateCheckout(ctx context.Context, req payment.CheckoutRequest) (*payment.CheckoutResult, error) {
	productID, ok := c.products[req.Product]
	if !ok {
		return nil, fmt.Errorf("dodo: no product ID configured for %s", req.Product)
	}
	metadata := map[string]string{
		"payment_intent_id": req.IntentID,
		"business_order_id": req.BusinessOrderID,
		"product":           string(req.Product),
	}
	for key, value := range req.Metadata {
		metadata[key] = value
	}

	params := dodopayments.CheckoutSessionNewParams{
		CheckoutSessionRequest: dodopayments.CheckoutSessionRequestParam{
			ProductCart: dodopayments.F([]dodopayments.ProductItemReqParam{{
				ProductID: dodopayments.F(productID),
				Quantity:  dodopayments.F(int64(1)),
				Amount:    dodopayments.F(req.AmountMinor),
			}}),
			Metadata:  dodopayments.F(metadata),
			ReturnURL: dodopayments.F(req.ReturnURL),
		},
	}
	if req.CancelURL != "" {
		params.CheckoutSessionRequest.CancelURL = dodopayments.F(req.CancelURL)
	}
	if req.Currency != "" {
		params.CheckoutSessionRequest.BillingCurrency = dodopayments.F(dodopayments.Currency(req.Currency))
	}
	resp, err := c.checkoutSvc.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("dodo: create checkout: %w", err)
	}
	return &payment.CheckoutResult{SessionID: resp.SessionID, CheckoutURL: resp.CheckoutURL}, nil
}

// VerifyWebhook verifies and normalizes a Dodo webhook event.
func (c *Client) VerifyWebhook(rawBody []byte, headers http.Header) (*payment.WebhookEvent, error) {
	event, err := c.webhookSvc.Unwrap(rawBody, headers)
	if err != nil {
		return nil, fmt.Errorf("dodo: verify webhook: %w", err)
	}

	union := event.AsUnion()
	paymentEvent, ok := union.(dodopayments.PaymentSucceededWebhookEvent)
	if !ok {
		return &payment.WebhookEvent{Provider: payment.ProviderDodo, Type: string(event.Type)}, nil
	}

	email := paymentEvent.Data.Customer.Email
	if email == "" {
		email = paymentEvent.Data.Metadata["email"]
	}
	return &payment.WebhookEvent{
		Provider: payment.ProviderDodo,
		Type:     string(event.Type),
		Data: payment.WebhookEventData{
			OrderID:   paymentEvent.Data.Metadata["payment_intent_id"],
			Amount:    int64(paymentEvent.Data.TotalAmount),
			Email:     email,
			PaymentID: paymentEvent.Data.PaymentID,
			Metadata:  paymentEvent.Data.Metadata,
		},
	}, nil
}

// WebhookAck returns an empty body; Dodo accepts any successful HTTP response.
func (c *Client) WebhookAck() []byte { return nil }

// FetchPayment reconciles one intent using Dodo's official lookup flow. A
// checkout session is resolved through GET /checkouts/{id}; its resulting
// payment is then loaded with GET /payments/{id} so amount and currency are
// always verified against the authoritative payment record.
func (c *Client) FetchPayment(ctx context.Context, intent payment.PaymentIntent) (*payment.WebhookEvent, error) {
	if intent.ProviderPaymentID == "" && intent.ProviderSessionID == "" {
		return nil, fmt.Errorf("dodo: intent %s has no checkout session or payment ID", intent.ID)
	}

	if intent.ProviderPaymentID != "" {
		resp, err := c.paymentsSvc.Get(ctx, intent.ProviderPaymentID)
		if err != nil {
			return nil, fmt.Errorf("dodo: fetch payment: %w", err)
		}
		if resp.Status != dodopayments.IntentStatusSucceeded {
			return nil, nil
		}
		if err := validateDodoPayment(intent, resp); err != nil {
			return nil, err
		}
		return dodoPaymentEvent(resp.Metadata, resp.PaymentID, resp.TotalAmount, string(resp.Currency), resp.Customer.Email), nil
	}

	session, err := c.checkoutSvc.Get(ctx, intent.ProviderSessionID)
	if err != nil {
		return nil, fmt.Errorf("dodo: fetch checkout session: %w", err)
	}

	if session.PaymentID == "" {
		return nil, nil
	}
	if session.PaymentStatus != dodopayments.IntentStatusSucceeded {
		return nil, nil
	}

	resp, err := c.paymentsSvc.Get(ctx, session.PaymentID)
	if err != nil {
		return nil, fmt.Errorf("dodo: fetch checkout payment: %w", err)
	}
	if resp.Status != dodopayments.IntentStatusSucceeded {
		return nil, nil
	}
	if err := validateDodoPayment(intent, resp); err != nil {
		return nil, err
	}
	return dodoPaymentEvent(resp.Metadata, resp.PaymentID, resp.TotalAmount, string(resp.Currency), resp.Customer.Email), nil
}

func validateDodoPayment(intent payment.PaymentIntent, resp *dodopayments.Payment) error {
	orderID := resp.Metadata["payment_intent_id"]
	if orderID == "" {
		return fmt.Errorf("dodo: payment %s has no payment_intent_id metadata", resp.PaymentID)
	}
	if orderID != intent.ID {
		return fmt.Errorf("dodo: payment %s metadata payment_intent_id %q does not match intent %q", resp.PaymentID, orderID, intent.ID)
	}
	if intent.ProviderSessionID != "" && resp.CheckoutSessionID != intent.ProviderSessionID {
		return fmt.Errorf("dodo: payment %s checkout session %q does not match intent %q", resp.PaymentID, resp.CheckoutSessionID, intent.ProviderSessionID)
	}
	if resp.TotalAmount != intent.AmountMinor {
		return fmt.Errorf("dodo: payment %s amount mismatch for intent %s: got %d want %d", resp.PaymentID, intent.ID, resp.TotalAmount, intent.AmountMinor)
	}
	if intent.Currency != "" && string(resp.Currency) != intent.Currency {
		return fmt.Errorf("dodo: payment %s currency mismatch for intent %s: got %s want %s", resp.PaymentID, intent.ID, resp.Currency, intent.Currency)
	}
	return nil
}

func dodoPaymentEvent(metadata map[string]string, paymentID string, amount int64, currency, email string) *payment.WebhookEvent {
	orderID := metadata["payment_intent_id"]
	if orderID == "" {
		orderID = metadata["business_order_id"]
	}
	if email == "" {
		email = metadata["email"]
	}
	return &payment.WebhookEvent{
		Provider: payment.ProviderDodo,
		Type:     payment.EventPaymentSucceeded,
		Data: payment.WebhookEventData{
			OrderID:   orderID,
			Amount:    amount,
			Currency:  currency,
			Email:     email,
			PaymentID: paymentID,
			Metadata:  metadata,
		},
	}
}
