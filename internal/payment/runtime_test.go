package payment

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
)

type fakeGateway struct {
	mu         sync.Mutex
	request    CheckoutRequest
	event      WebhookEvent
	fetchEvent WebhookEvent
	fetchID    string
	err        error
}

func (g *fakeGateway) CreateCheckout(ctx context.Context, req CheckoutRequest) (*CheckoutResult, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.request = req
	if g.err != nil {
		return nil, g.err
	}
	return &CheckoutResult{SessionID: "session", CheckoutURL: "https://pay.example/checkout"}, nil
}

func (g *fakeGateway) VerifyWebhook(raw []byte, header http.Header) (*WebhookEvent, error) {
	return &g.event, g.err
}

func (g *fakeGateway) WebhookAck() []byte { return []byte("ok") }

func (g *fakeGateway) FetchPayment(ctx context.Context, intent PaymentIntent) (*WebhookEvent, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.fetchID = intent.ID
	event := g.fetchEvent
	return &event, g.err
}

type fakeFlow struct {
	product string
	order   BusinessOrder
	onPaid  int
}

func (f *fakeFlow) Product() string { return f.product }

func (f *fakeFlow) PrepareCheckout(ctx context.Context, businessOrderID string) (BusinessOrder, error) {
	if businessOrderID != f.order.ID {
		return BusinessOrder{}, errors.New("order not found")
	}
	return f.order, nil
}

func (f *fakeFlow) OnPaid(ctx context.Context, intent PaymentIntent) error {
	f.onPaid++
	return nil
}

func (f *fakeFlow) SuccessRedirect(intent PaymentIntent) string {
	return "/zh-Hant/naming/chat?order_id=" + intent.BusinessOrderID
}

func (f *fakeFlow) FailureRedirect(intent PaymentIntent) string {
	return "/zh-Hant/naming?order_id=" + intent.BusinessOrderID
}

type fakeSender struct {
	messages []EmailMessage
}

type failingFlow struct {
	fakeFlow
	onPaidErr error
}

func (f *failingFlow) OnPaid(ctx context.Context, intent PaymentIntent) error {
	_ = intent
	return f.onPaidErr
}

func (s *fakeSender) Send(ctx context.Context, msg EmailMessage) error {
	s.messages = append(s.messages, msg)
	return nil
}

func newRuntimeTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "payments.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, migrate := range []func(context.Context) error{
		NewIntentStore(db).Migrate,
		NewNotificationStore(db).Migrate,
		NewFulfillmentTaskStore(db).Migrate,
		NewPaymentEventStore(db).Migrate,
	} {
		if err := migrate(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func newTestRuntime(t *testing.T) (*Runtime, *fakeGateway, *fakeFlow, *fakeSender, *IntentStore, *NotificationStore) {
	t.Helper()
	db := newRuntimeTestDB(t)
	gateway := &fakeGateway{}
	flow := &fakeFlow{
		product: "naming",
		order: BusinessOrder{
			ID: "biz-1", AmountMinor: 990, Currency: "CNY",
			Email: "user@example.com", Locale: "zh-Hant",
		},
	}
	sender := &fakeSender{}
	intents := NewIntentStore(db)
	notifications := NewNotificationStore(db)
	runtime := NewRuntime(
		map[Provider]Gateway{ProviderXunhu: gateway},
		map[string]ProductFlow{"naming": flow},
		intents,
		notifications,
		NewFulfillmentTaskStore(db),
		NewPaymentEventStore(db),
		sender,
		RuntimeOptions{AdminEmail: "admin@example.com", BaseURL: "https://liki.hk"},
	)
	return runtime, gateway, flow, sender, intents, notifications
}

func checkoutEvent(intent PaymentIntent) WebhookEvent {
	return WebhookEvent{
		Provider: ProviderXunhu,
		Type:     EventPaymentSucceeded,
		Data: WebhookEventData{
			OrderID:   intent.ID,
			Amount:    intent.AmountMinor,
			Email:     intent.Email,
			PaymentID: "pay-1",
		},
	}
}

func TestRuntimeCreateIntentUsesProductFlow(t *testing.T) {
	ctx := context.Background()
	runtime, _, flow, _, store, _ := newTestRuntime(t)

	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{
		Product:         "naming",
		BusinessOrderID: "biz-1",
		Provider:        ProviderXunhu,
	})
	if err != nil {
		t.Fatal(err)
	}
	if intent.ID == "" {
		t.Fatal("intent id is empty")
	}
	if intent.AmountMinor != 990 || intent.Currency != "CNY" || intent.Email != "user@example.com" {
		t.Fatalf("intent = %+v", intent)
	}
	got, err := store.GetIntent(ctx, intent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != IntentCreated || got.Product != "naming" {
		t.Fatalf("stored intent = %+v", got)
	}
	if got.SuccessURL != intent.SuccessURL || got.FailureURL != intent.FailureURL {
		t.Fatalf("stored redirects = success %q failure %q, want success %q failure %q",
			got.SuccessURL, got.FailureURL, intent.SuccessURL, intent.FailureURL)
	}
	if flow.order.ID != "biz-1" {
		t.Fatal("product flow was not consulted")
	}
}

func TestRuntimeStartCheckoutSendsReturnURLAndStoresSession(t *testing.T) {
	ctx := context.Background()
	runtime, gateway, _, _, store, _ := newTestRuntime(t)
	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{
		Product: "naming", BusinessOrderID: "biz-1", Provider: ProviderXunhu,
	})
	if err != nil {
		t.Fatal(err)
	}

	result, _, err := runtime.StartCheckout(ctx, intent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gateway.request.IntentID != intent.ID || gateway.request.BusinessOrderID != "biz-1" {
		t.Fatalf("gateway request = %+v", gateway.request)
	}
	if want := "https://liki.hk/api/payments/return/" + intent.ID; gateway.request.ReturnURL != want {
		t.Fatalf("return URL = %q, want %q", gateway.request.ReturnURL, want)
	}
	if result.CheckoutURL == "" {
		t.Fatal("checkout URL is empty")
	}
	got, err := store.GetIntent(ctx, intent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != IntentCheckoutCreated || got.CheckoutURL != result.CheckoutURL {
		t.Fatalf("intent after checkout = %+v", got)
	}
}

func TestRuntimeWebhookMarksPaidDispatchesOnceAndQueuesNotifications(t *testing.T) {
	ctx := context.Background()
	runtime, gateway, flow, sender, _, notifications := newTestRuntime(t)
	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{
		Product: "naming", BusinessOrderID: "biz-1", Provider: ProviderXunhu,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = runtime.StartCheckout(ctx, intent.ID); err != nil {
		t.Fatal(err)
	}

	raw := []byte("provider=webhook")
	gateway.event = checkoutEvent(intent)
	if err := runtime.HandleWebhookEvent(ctx, raw, nil); err != nil {
		t.Fatal(err)
	}
	if flow.onPaid != 0 {
		t.Fatalf("OnPaid count before worker = %d, want 0", flow.onPaid)
	}
	if n, err := runtime.ProcessFulfillmentTasks(ctx, 10); err != nil || n != 1 {
		t.Fatalf("ProcessFulfillmentTasks = %d, %v; want 1, nil", n, err)
	}
	if flow.onPaid != 1 {
		t.Fatalf("OnPaid count = %d, want 1", flow.onPaid)
	}
	queued, err := notifications.Pending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(queued) != 2 {
		t.Fatalf("queued notifications = %d, want customer and admin", len(queued))
	}
	if queued[0].Kind != NotificationCustomerReceipt {
		t.Fatalf("first notification = %q, want customer receipt", queued[0].Kind)
	}
	if body := queued[0].Body; !strings.Contains(body, "付款已完成") || strings.Contains(body, "服务已开通") {
		t.Fatalf("customer receipt should confirm payment without claiming fulfillment: %q", body)
	}
	if err := runtime.ProcessPendingNotifications(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if len(sender.messages) != 2 {
		t.Fatalf("messages = %d, want customer and admin", len(sender.messages))
	}

	if err := runtime.HandleWebhookEvent(ctx, raw, nil); err != nil {
		t.Fatal(err)
	}
	if n, err := runtime.ProcessFulfillmentTasks(ctx, 10); err != nil || n != 0 {
		t.Fatalf("duplicate fulfillment process = %d, %v; want 0, nil", n, err)
	}
	if flow.onPaid != 1 {
		t.Fatalf("duplicate webhook OnPaid count = %d, want 1", flow.onPaid)
	}
}

func TestRuntimeWebhookRejectsAmountMismatch(t *testing.T) {
	ctx := context.Background()
	runtime, gateway, flow, _, _, _ := newTestRuntime(t)
	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{
		Product: "naming", BusinessOrderID: "biz-1", Provider: ProviderXunhu,
	})
	if err != nil {
		t.Fatal(err)
	}
	gateway.event = checkoutEvent(intent)
	gateway.event.Data.Amount = int64(1)

	if err := runtime.HandleWebhookEvent(ctx, []byte("x"), nil); err == nil {
		t.Fatal("expected amount mismatch error")
	}
	if flow.onPaid != 0 {
		t.Fatalf("OnPaid count = %d, want 0", flow.onPaid)
	}
}

func TestRuntimeCaptureQueuesNotificationsBeforeProductFulfillment(t *testing.T) {
	ctx := context.Background()
	runtime, gateway, _, sender, _, notifications := newTestRuntime(t)
	flow := &failingFlow{
		fakeFlow: fakeFlow{
			product: "naming",
			order: BusinessOrder{
				ID: "biz-1", AmountMinor: 990, Currency: "CNY",
				Email: "user@example.com", Locale: "zh-Hant",
			},
		},
		onPaidErr: errors.New("report generator failed"),
	}
	runtime.flows["naming"] = flow

	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{
		Product: "naming", BusinessOrderID: "biz-1", Provider: ProviderXunhu,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := runtime.StartCheckout(ctx, intent.ID); err != nil {
		t.Fatal(err)
	}
	gateway.event = checkoutEvent(intent)
	if err := runtime.HandleWebhookEvent(ctx, []byte("event"), nil); err != nil {
		t.Fatal(err)
	}

	queued, err := notifications.Pending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(queued) != 2 {
		t.Fatalf("queued notifications = %d, want customer and admin", len(queued))
	}
	if err := runtime.ProcessPendingNotifications(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if len(sender.messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(sender.messages))
	}
}

func TestRuntimeReconcilePendingIntentMarksPaidOnce(t *testing.T) {
	ctx := context.Background()
	runtime, gateway, flow, _, store, notifications := newTestRuntime(t)
	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{
		Product: "naming", BusinessOrderID: "biz-1", Provider: ProviderXunhu,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := runtime.StartCheckout(ctx, intent.ID); err != nil {
		t.Fatal(err)
	}
	gateway.fetchEvent = checkoutEvent(intent)

	if err := runtime.ReconcilePendingIntents(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if gateway.fetchID != intent.ID {
		t.Fatalf("fetched intent = %q, want %q", gateway.fetchID, intent.ID)
	}
	if n, processErr := runtime.ProcessFulfillmentTasks(ctx, 10); processErr != nil || n != 1 {
		t.Fatalf("ProcessFulfillmentTasks = %d, %v; want 1, nil", n, processErr)
	}
	if flow.onPaid != 1 {
		t.Fatalf("OnPaid count = %d, want 1", flow.onPaid)
	}
	queued, err := notifications.Pending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(queued) != 2 {
		t.Fatalf("queued notifications = %d, want 2", len(queued))
	}

	gateway.fetchID = ""
	if err := runtime.ReconcilePendingIntents(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if gateway.fetchID != "" {
		t.Fatalf("paid intent was fetched again: %q", gateway.fetchID)
	}
	if flow.onPaid != 1 {
		t.Fatalf("OnPaid count after reconcile = %d, want 1", flow.onPaid)
	}
	got, err := store.GetIntent(ctx, intent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != IntentPaid {
		t.Fatalf("status = %q", got.Status)
	}
}

func TestRuntimeReconcileIgnoresUnpaidProviderState(t *testing.T) {
	ctx := context.Background()
	runtime, gateway, flow, _, store, _ := newTestRuntime(t)
	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{
		Product: "naming", BusinessOrderID: "biz-1", Provider: ProviderXunhu,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := runtime.StartCheckout(ctx, intent.ID); err != nil {
		t.Fatal(err)
	}
	gateway.fetchEvent = WebhookEvent{
		Provider: ProviderXunhu,
		Type:     "pending",
		Data: WebhookEventData{
			OrderID: intent.ID, Amount: intent.AmountMinor, Currency: intent.Currency,
		},
	}

	if err := runtime.ReconcilePendingIntents(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if flow.onPaid != 0 {
		t.Fatalf("OnPaid count = %d, want 0", flow.onPaid)
	}
	got, err := store.GetIntent(ctx, intent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != IntentCheckoutCreated {
		t.Fatalf("status = %q", got.Status)
	}
}

func TestRuntimeWebhookDoesNotGrantUnifiedChatEntitlement(t *testing.T) {
	ctx := context.Background()
	runtime, gateway, _, _, _, _ := newTestRuntime(t)
	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{
		Product: "naming", BusinessOrderID: "biz-1", Provider: ProviderXunhu,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := runtime.StartCheckout(ctx, intent.ID); err != nil {
		t.Fatal(err)
	}
	gateway.event = checkoutEvent(intent)
	if err := runtime.HandleWebhookEvent(ctx, []byte("event"), nil); err != nil {
		t.Fatal(err)
	}

	if err := runtime.HandleWebhookEvent(ctx, []byte("event-duplicate"), nil); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeReturnReconcilesCompletedProviderPayment(t *testing.T) {
	ctx := context.Background()
	runtime, gateway, flow, _, _, _ := newTestRuntime(t)
	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{
		Product: "naming", BusinessOrderID: "biz-1", Provider: ProviderXunhu,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := runtime.StartCheckout(ctx, intent.ID); err != nil {
		t.Fatal(err)
	}
	gateway.fetchEvent = checkoutEvent(intent)

	returned, location, err := runtime.HandleReturn(ctx, intent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if returned.Status != IntentPaid || !strings.Contains(location, "payment_intent_id=") {
		t.Fatalf("return = %+v, %q", returned, location)
	}

	if _, processErr := runtime.ProcessFulfillmentTasks(ctx, 10); processErr != nil || flow.onPaid != 1 {
		t.Fatalf("ProcessFulfillmentTasks = %d, %v; want 1, nil", flow.onPaid, processErr)
	}
}
