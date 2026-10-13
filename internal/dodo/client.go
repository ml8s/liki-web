package dodo

import (
	"context"
	"fmt"
	"maps"
	"net/http"

	dodopayments "github.com/dodopayments/dodopayments-go"
	"github.com/dodopayments/dodopayments-go/option"

	"github.com/ml8s/liki-web/internal/payment"
)

// Client wraps the Dodo Payments SDK for checkout and webhook operations.
type Client struct {
	checkoutSvc     *dodopayments.CheckoutSessionService
	subscriptionSvc *dodopayments.SubscriptionService
	webhookSvc      *dodopayments.WebhookService
	paymentsSvc     *dodopayments.PaymentService
	products        map[string]string
}

// New creates a Dodo Payments client with the given API key and webhook secret.
func New(apiKey, webhookKey string, testMode bool, products map[string]string) *Client {
	opts := []option.RequestOption{option.WithBearerToken(apiKey)}
	if testMode {
		opts = append(opts, option.WithEnvironmentTestMode())
	} else {
		opts = append(opts, option.WithEnvironmentLiveMode())
	}
	return &Client{
		checkoutSvc:     dodopayments.NewCheckoutSessionService(opts...),
		subscriptionSvc: dodopayments.NewSubscriptionService(opts...),
		webhookSvc:      dodopayments.NewWebhookService(append(opts, option.WithWebhookKey(webhookKey))...),
		paymentsSvc:     dodopayments.NewPaymentService(opts...),
		products:        products,
	}
}

// CreateCheckout creates a Dodo checkout session from the normalized request.
func (c *Client) CreateCheckout(ctx context.Context, req payment.CheckoutRequest) (*payment.CheckoutResult, error) {
	productID, ok := c.products[req.Product+":one_time"]
	if !ok || productID == "" {
		return nil, fmt.Errorf("dodo: no product ID configured for %s", req.Product)
	}
	metadata := map[string]string{
		"payment_intent_id": req.IntentID,
		"business_order_id": req.BusinessOrderID,
		"product":           string(req.Product),
	}
	maps.Copy(metadata, req.Metadata)

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

// CreateSubscriptionCheckout creates a hosted recurring checkout. The amount is
// intentionally omitted: a recurring Dodo product owns its price.
func (c *Client) CreateSubscriptionCheckout(ctx context.Context, req payment.SubscriptionCheckoutRequest) (*payment.CheckoutResult, error) {
	productKey := req.Product + ":subscription"
	productID, ok := c.products[productKey]
	if !ok || productID == "" {
		return nil, fmt.Errorf("dodo: no subscription product ID configured for %s", req.Product)
	}
	metadata := map[string]string{
		"liki_product":          req.Product,
		"liki_billing_mode":     "subscription",
		"liki_user_id":          req.UserID,
		"liki_subscription_key": req.SubscriptionKey,
	}
	params := dodopayments.CheckoutSessionNewParams{
		CheckoutSessionRequest: dodopayments.CheckoutSessionRequestParam{
			ProductCart: dodopayments.F([]dodopayments.ProductItemReqParam{{
				ProductID: dodopayments.F(productID),
				Quantity:  dodopayments.F(int64(1)),
			}}),
			Metadata:        dodopayments.F(metadata),
			ReturnURL:       dodopayments.F(req.ReturnURL),
			CancelURL:       dodopayments.F(req.CancelURL),
			BillingCurrency: dodopayments.F(dodopayments.Currency(req.Currency)),
		},
	}
	resp, err := c.checkoutSvc.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("dodo: create subscription checkout: %w", err)
	}
	return &payment.CheckoutResult{SessionID: resp.SessionID, CheckoutURL: resp.CheckoutURL}, nil
}

// SetSubscriptionCancellation schedules or cancels termination at period end.
func (c *Client) SetSubscriptionCancellation(ctx context.Context, subscriptionID string, cancel bool) error {
	_, err := c.subscriptionSvc.Update(ctx, subscriptionID, dodopayments.SubscriptionUpdateParams{
		CancelAtNextBillingDate: dodopayments.F(cancel),
	})
	if err != nil {
		return fmt.Errorf("dodo: update subscription cancellation: %w", err)
	}
	return nil
}

// FetchSubscriptionByCheckout heals a lost subscription webhook by following
// Dodo's authoritative checkout → payment → subscription lookup chain.
func (c *Client) FetchSubscriptionByCheckout(ctx context.Context, checkoutID string) (*payment.WebhookEvent, error) {
	session, err := c.checkoutSvc.Get(ctx, checkoutID)
	if err != nil {
		return nil, fmt.Errorf("dodo: fetch subscription checkout: %w", err)
	}
	if session.PaymentID == "" || session.PaymentStatus != dodopayments.IntentStatusSucceeded {
		return nil, nil
	}
	paymentRecord, err := c.paymentsSvc.Get(ctx, session.PaymentID)
	if err != nil {
		return nil, fmt.Errorf("dodo: fetch subscription checkout payment: %w", err)
	}
	if paymentRecord.SubscriptionID == "" || paymentRecord.Status != dodopayments.IntentStatusSucceeded {
		return nil, nil
	}
	subscription, err := c.subscriptionSvc.Get(ctx, paymentRecord.SubscriptionID)
	if err != nil {
		return nil, fmt.Errorf("dodo: fetch reconciled subscription: %w", err)
	}
	eventType := payment.EventSubscriptionUpdated
	switch subscription.Status {
	case dodopayments.SubscriptionStatusActive:
		eventType = payment.EventSubscriptionActive
	case dodopayments.SubscriptionStatusOnHold:
		eventType = payment.EventSubscriptionOnHold
	case dodopayments.SubscriptionStatusCancelled:
		eventType = payment.EventSubscriptionCancelled
	case dodopayments.SubscriptionStatusExpired:
		eventType = payment.EventSubscriptionExpired
	case dodopayments.SubscriptionStatusFailed:
		eventType = payment.EventSubscriptionFailed
	}
	return normalizeSubscription(eventType, *subscription, subscription.CreatedAt), nil
}

// VerifyWebhook verifies and normalizes a Dodo webhook event.
func (c *Client) VerifyWebhook(rawBody []byte, headers http.Header) (*payment.WebhookEvent, error) {
	event, err := c.webhookSvc.Unwrap(rawBody, headers)
	if err != nil {
		return nil, fmt.Errorf("dodo: verify webhook: %w", err)
	}

	union := event.AsUnion()
	if subscriptionEvent, ok := union.(dodopayments.SubscriptionActiveWebhookEvent); ok {
		return normalizeSubscription(string(subscriptionEvent.Type), subscriptionEvent.Data, subscriptionEvent.Timestamp), nil
	}
	if subscriptionEvent, ok := union.(dodopayments.SubscriptionRenewedWebhookEvent); ok {
		return normalizeSubscription(string(subscriptionEvent.Type), subscriptionEvent.Data, subscriptionEvent.Timestamp), nil
	}
	if subscriptionEvent, ok := union.(dodopayments.SubscriptionUpdatedWebhookEvent); ok {
		return normalizeSubscription(string(subscriptionEvent.Type), subscriptionEvent.Data, subscriptionEvent.Timestamp), nil
	}
	if subscriptionEvent, ok := union.(dodopayments.SubscriptionOnHoldWebhookEvent); ok {
		return normalizeSubscription(string(subscriptionEvent.Type), subscriptionEvent.Data, subscriptionEvent.Timestamp), nil
	}
	if subscriptionEvent, ok := union.(dodopayments.SubscriptionFailedWebhookEvent); ok {
		return normalizeSubscription(string(subscriptionEvent.Type), subscriptionEvent.Data, subscriptionEvent.Timestamp), nil
	}
	if subscriptionEvent, ok := union.(dodopayments.SubscriptionCancelledWebhookEvent); ok {
		return normalizeSubscription(string(subscriptionEvent.Type), subscriptionEvent.Data, subscriptionEvent.Timestamp), nil
	}
	if subscriptionEvent, ok := union.(dodopayments.SubscriptionExpiredWebhookEvent); ok {
		return normalizeSubscription(string(subscriptionEvent.Type), subscriptionEvent.Data, subscriptionEvent.Timestamp), nil
	}
	if refundEvent, ok := union.(dodopayments.RefundSucceededWebhookEvent); ok {
		return normalizeRefund(refundEvent.Type, refundEvent.Data, refundEvent.Timestamp), nil
	}
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
			OrderID:           paymentEvent.Data.Metadata["payment_intent_id"],
			Amount:            int64(paymentEvent.Data.TotalAmount),
			Email:             email,
			PaymentID:         paymentEvent.Data.PaymentID,
			Metadata:          paymentEvent.Data.Metadata,
			SubscriptionID:    paymentEvent.Data.SubscriptionID,
			ProviderEventTime: timePtr(paymentEvent.Timestamp),
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
