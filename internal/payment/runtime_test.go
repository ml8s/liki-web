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
	return &CheckoutResult{CheckoutSessionID: "session", CheckoutURL: "https://pay.example/checkout"}, nil
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

func (f *fakeFlow) DisplayName(locale string) string {
	if locale == "en" {
		return "Test Product"
	}
	return "测试产品"
}

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
	return "/app/chat?order_id=" + intent.BusinessOrderID
}

func (f *fakeFlow) FailureRedirect(intent PaymentIntent) string {
	return "/app?order_id=" + intent.BusinessOrderID
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
		NewPaymentCompletionTaskStore(db).Migrate,
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
		product: "test-product",
		order: BusinessOrder{
			ID: "biz-1", AmountMinor: 990, Currency: "CNY",
			Email: "user@example.com", Locale: "zh-Hant",
		},
	}
	sender := &fakeSender{}
	intents := NewIntentStore(db)
	notifications := NewNotificationStore(db)
	runtime := NewRuntime(
		map[Provider]Gateway{ProviderXunhu: gateway, ProviderDodo: gateway},
		map[string]ProductFlow{"test-product": flow},
		intents,
		notifications,
		NewPaymentCompletionTaskStore(db),
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

func TestRuntimeRoutesUSDToDodoDespiteHint(t *testing.T) {
	ctx := context.Background()
	runtime, _, flow, _, _, _ := newTestRuntime(t)
	flow.order.Currency = "USD"

	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{
		Product:         "test-product",
		BusinessOrderID: "biz-1",
		Provider:        ProviderXunhu,
	})
	if err != nil {
		t.Fatal(err)
	}
	if intent.Provider != ProviderDodo {
		t.Fatalf("provider = %q, want dodo", intent.Provider)
	}
}

func TestQueuePaidNotificationsUsesUnifiedTemplateData(t *testing.T) {
	ctx := context.Background()
	runtime, gateway, flow, _, intents, notifications := newTestRuntime(t)
	flow.order = BusinessOrder{
		ID: "order-<1>", AmountMinor: 100, Currency: "USD",
		Email: "user@example.com", Locale: "zh-Hant",
	}
	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{
		Product: "test-product", BusinessOrderID: "order-<1>",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = runtime.StartCheckout(ctx, intent.ID); err != nil {
		t.Fatal(err)
	}
	gateway.event = WebhookEvent{
		Provider: ProviderDodo,
		Type:     EventPaymentSucceeded,
		Data: WebhookEventData{
			OrderID: intent.ID, Amount: 100, Currency: "USD",
			PaymentID: "pay-1",
		},
	}
	if err := runtime.HandleWebhookEvent(ctx, []byte("event"), nil); err != nil {
		t.Fatal(err)
	}
	updated, err := intents.GetIntent(ctx, intent.ID)
	if err != nil {
		t.Fatal(err)
	}
	queued, err := notifications.Pending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(queued) != 2 {
		t.Fatalf("queued notifications = %d, want two", len(queued))
	}

	customer := queued[0]
	if customer.Kind != NotificationCustomerReceipt || customer.To != "user@example.com" {
		t.Fatalf("customer notification = %+v", customer)
	}
	for _, want := range []string{
		"测试产品", "order-&lt;1&gt;", "$1.00", "https://liki.hk/app/chat",
		"https://liki.hk</a> 使用 Liki",
	} {
		if !strings.Contains(customer.Body, want) {
			t.Fatalf("customer body %q missing %q", customer.Body, want)
		}
	}
	if strings.Contains(customer.Body, "服务已开通") || strings.Contains(customer.Body, "990 CNY") {
		t.Fatalf("customer body contains invalid fulfillment/raw amount text: %q", customer.Body)
	}

	admin := queued[1]
	if admin.Kind != NotificationAdminNewOrder || admin.To != "admin@example.com" {
		t.Fatalf("admin notification = %+v", admin)
	}
	for _, want := range []string{
		"测试产品", intent.ID, "user@example.com", "$1.00", "Dodo",
		updated.PaidAt.UTC().Format("2006-01-02 15:04:05 MST"),
		"https://liki.hk</a> 使用 Liki",
	} {
		if !strings.Contains(admin.Body, want) {
			t.Fatalf("admin body %q missing %q", admin.Body, want)
		}
	}
}

func TestQueuePaidNotificationsLocalizesCustomerReceipt(t *testing.T) {
	ctx := context.Background()
	runtime, gateway, flow, _, _, notifications := newTestRuntime(t)
	flow.order = BusinessOrder{
		ID: "order-en", AmountMinor: 100, Currency: "USD",
		Email: "user@example.com", Locale: "en",
	}
	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{
		Product: "test-product", BusinessOrderID: "order-en",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = runtime.StartCheckout(ctx, intent.ID); err != nil {
		t.Fatal(err)
	}
	gateway.event = WebhookEvent{
		Provider: ProviderDodo,
		Type:     EventPaymentSucceeded,
		Data: WebhookEventData{
			OrderID: intent.ID, Amount: 100, Currency: "USD",
			PaymentID: "pay-en",
		},
	}
	if err := runtime.HandleWebhookEvent(ctx, []byte("event"), nil); err != nil {
		t.Fatal(err)
	}

	queued, err := notifications.Pending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(queued) != 2 {
		t.Fatalf("queued notifications = %d, want two", len(queued))
	}

	customer := queued[0]
	for _, want := range []string{
		"Payment received", "Test Product", "Order: order-en", "$1.00", "Continue",
	} {
		if !strings.Contains(customer.Body, want) {
			t.Fatalf("English customer body %q missing %q", customer.Body, want)
		}
	}

	admin := queued[1]
	for _, want := range []string{
		"新订单", "Test Product", "user@example.com", "$1.00", "Dodo",
	} {
		if !strings.Contains(admin.Body, want) {
			t.Fatalf("Chinese admin body %q missing %q", admin.Body, want)
		}
	}
}

func TestRuntimeCreateIntentUsesProductFlow(t *testing.T) {
	ctx := context.Background()
	runtime, _, flow, _, store, _ := newTestRuntime(t)

	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{
		Product:         "test-product",
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
	if got.Status != IntentCreated || got.Product != "test-product" {
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
		Product: "test-product", BusinessOrderID: "biz-1", Provider: ProviderXunhu,
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
	// 产品流只回站内路径，交给网关的必须是绝对地址（Dodo 对相对 cancel_url 直接 400）。
	if want := "https://liki.hk/app?order_id=biz-1"; gateway.request.CancelURL != want {
		t.Fatalf("cancel URL = %q, want %q", gateway.request.CancelURL, want)
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

func TestCheckoutRequestRejectsRelativeURLs(t *testing.T) {
	for _, tc := range []struct {
		name      string
		returnURL string
		cancelURL string
		wantErr   bool
	}{
		{name: "absolute", returnURL: "https://liki.hk/api/payments/return/1", cancelURL: "https://liki.hk/app/orders"},
		{name: "empty omits", returnURL: "https://liki.hk/api/payments/return/1"},
		{name: "relative cancel", returnURL: "https://liki.hk/api/payments/return/1", cancelURL: "/app/orders?status=unpaid", wantErr: true},
		{name: "relative return", returnURL: "/api/payments/return/1", cancelURL: "https://liki.hk/app/orders", wantErr: true},
		{name: "other scheme", returnURL: "https://liki.hk/api/payments/return/1", cancelURL: "javascript:alert(1)", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckoutRequest{ReturnURL: tc.returnURL, CancelURL: tc.cancelURL}.validateURLs()
			if tc.wantErr && err == nil {
				t.Fatal("validateURLs() accepted an invalid URL")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("validateURLs() = %v", err)
			}
		})
	}
}

func TestRuntimeWebhookMarksPaidDispatchesOnceAndQueuesNotifications(t *testing.T) {
	ctx := context.Background()
	runtime, gateway, flow, sender, _, notifications := newTestRuntime(t)
	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{
		Product: "test-product", BusinessOrderID: "biz-1", Provider: ProviderXunhu,
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
	if n, err := runtime.ProcessPaymentCompletionTasks(ctx, 10); err != nil || n != 1 {
		t.Fatalf("ProcessPaymentCompletionTasks = %d, %v; want 1, nil", n, err)
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
	if n, err := runtime.ProcessPaymentCompletionTasks(ctx, 10); err != nil || n != 0 {
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
		Product: "test-product", BusinessOrderID: "biz-1", Provider: ProviderXunhu,
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
			product: "test-product",
			order: BusinessOrder{
				ID: "biz-1", AmountMinor: 990, Currency: "CNY",
				Email: "user@example.com", Locale: "zh-Hant",
			},
		},
		onPaidErr: errors.New("report generator failed"),
	}
	runtime.flows["test-product"] = flow

	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{
		Product: "test-product", BusinessOrderID: "biz-1", Provider: ProviderXunhu,
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
		Product: "test-product", BusinessOrderID: "biz-1", Provider: ProviderXunhu,
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
	if n, processErr := runtime.ProcessPaymentCompletionTasks(ctx, 10); processErr != nil || n != 1 {
		t.Fatalf("ProcessPaymentCompletionTasks = %d, %v; want 1, nil", n, processErr)
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
		Product: "test-product", BusinessOrderID: "biz-1", Provider: ProviderXunhu,
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

func TestRuntimeDuplicateWebhookIsIdempotent(t *testing.T) {
	ctx := context.Background()
	runtime, gateway, _, _, _, _ := newTestRuntime(t)
	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{
		Product: "test-product", BusinessOrderID: "biz-1", Provider: ProviderXunhu,
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
		Product: "test-product", BusinessOrderID: "biz-1", Provider: ProviderXunhu,
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

	if _, processErr := runtime.ProcessPaymentCompletionTasks(ctx, 10); processErr != nil || flow.onPaid != 1 {
		t.Fatalf("ProcessPaymentCompletionTasks = %d, %v; want 1, nil", flow.onPaid, processErr)
	}
}

func TestPaymentCompletionObserverCannotFailProductFulfillment(t *testing.T) {
	ctx := context.Background()
	runtime, gateway, flow, _, _, _ := newTestRuntime(t)
	observed := 0
	runtime.SetPaymentCompletionObserver(func(ctx context.Context, intent PaymentIntent) {
		observed++
		if intent.Provider != ProviderXunhu {
			t.Errorf("provider = %q, want xunhu", intent.Provider)
		}
	})
	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{Product: "test-product", BusinessOrderID: "biz-1", Provider: ProviderXunhu})
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
	if n, err := runtime.ProcessPaymentCompletionTasks(ctx, 10); err != nil || n != 1 {
		t.Fatalf("process = %d, %v", n, err)
	}
	if flow.onPaid != 1 || observed != 1 {
		t.Fatalf("flow/observer = %d/%d, want 1/1", flow.onPaid, observed)
	}
	task, ok, err := runtime.GetPaymentCompletionStatus(ctx, intent.ID)
	if err != nil || !ok || task.Status != PaymentCompletionSucceeded {
		t.Fatalf("task = %+v, %v, %v", task, ok, err)
	}
}

func TestRefundWebhookMarksIntentAndNotifiesObserver(t *testing.T) {
	ctx := context.Background()
	runtime, gateway, flow, _, _, _ := newTestRuntime(t)
	refunds := 0
	runtime.SetRefundObserver(func(ctx context.Context, event WebhookEvent, intent PaymentIntent) error {
		refunds++
		if event.Type != EventPaymentRefunded || intent.Status != PaymentStatusRefunded {
			t.Errorf("observer received event=%q status=%q", event.Type, intent.Status)
		}
		return nil
	})
	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{Product: "test-product", BusinessOrderID: "biz-1", Provider: ProviderXunhu})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := runtime.StartCheckout(ctx, intent.ID); err != nil {
		t.Fatal(err)
	}
	gateway.event = checkoutEvent(intent)
	if err := runtime.HandleWebhookEvent(ctx, []byte("paid"), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.ProcessPaymentCompletionTasks(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.ProcessPaymentCompletionTasks(ctx, 10); err != nil {
		t.Fatal(err)
	}

	refund := checkoutEvent(intent)
	refund.Type = EventPaymentRefunded
	gateway.event = refund
	if err := runtime.HandleWebhookEvent(ctx, []byte("refund"), nil); err != nil {
		t.Fatalf("refund webhook: %+v", err)
	}
	got, err := storeGetIntent(runtime, intent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != PaymentStatusRefunded {
		t.Fatalf("status = %q, want refunded", got.Status)
	}

	// A duplicate refund webhook must remain idempotent and must not re-run the
	// downstream observer.
	if err := runtime.HandleWebhookEvent(ctx, []byte("refund"), nil); err != nil {
		t.Fatal(err)
	}
	if refunds != 1 || flow.onPaid != 1 {
		t.Fatalf("refunds=%d onPaid=%d, want 1/1", refunds, flow.onPaid)
	}
}

func TestRefundReferencePrefersExplicitRefundID(t *testing.T) {
	tests := []struct {
		name string
		data WebhookEventData
		want string
	}{
		{name: "explicit", data: WebhookEventData{RefundID: "refund", PaymentID: "pay"}, want: "refund"},
		{name: "legacy", data: WebhookEventData{PaymentID: "refund"}, want: "refund"},
		{name: "missing", data: WebhookEventData{}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.data.RefundReference(); got != tt.want {
				t.Fatalf("RefundReference() = %q, want %q", got, tt.want)
			}
		})
	}
}

func storeGetIntent(runtime *Runtime, id string) (PaymentIntent, error) {
	return runtime.GetIntent(context.Background(), id)
}

func TestPartialRefundWebhookRecordsLedgerAndKeepsIntentPaid(t *testing.T) {
	ctx := context.Background()
	runtime, gateway, _, _, _, _ := newTestRuntime(t)
	events := make([]PaymentIntent, 0, 2)
	runtime.SetRefundObserver(func(ctx context.Context, event WebhookEvent, intent PaymentIntent) error {
		events = append(events, intent)
		return nil
	})
	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{Product: "test-product", BusinessOrderID: "biz-1", Provider: ProviderXunhu})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := runtime.StartCheckout(ctx, intent.ID); err != nil {
		t.Fatal(err)
	}
	gateway.event = checkoutEvent(intent)
	if err := runtime.HandleWebhookEvent(ctx, []byte("paid"), nil); err != nil {
		t.Fatal(err)
	}

	refund := checkoutEvent(intent)
	refund.Type = EventPaymentRefunded
	refund.Data.PaymentID = "refund-partial"
	refund.Data.Amount = 400
	gateway.event = refund
	if err := runtime.HandleWebhookEvent(ctx, []byte("partial"), nil); err != nil {
		t.Fatal(err)
	}
	got, err := runtime.GetIntent(ctx, intent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != IntentPaid || got.RefundedAmountMinor != 400 || len(events) != 1 {
		t.Fatalf("partial intent=%+v events=%d", got, len(events))
	}

	// The same provider refund ID is at-least-once by nature.
	if err := runtime.HandleWebhookEvent(ctx, []byte("partial"), nil); err != nil {
		t.Fatal(err)
	}
	refund.Data.PaymentID = "refund-final"
	refund.Data.Amount = 590
	gateway.event = refund
	if err := runtime.HandleWebhookEvent(ctx, []byte("final"), nil); err != nil {
		t.Fatal(err)
	}
	got, err = runtime.GetIntent(ctx, intent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != PaymentStatusRefunded || got.RefundedAmountMinor != 990 {
		t.Fatalf("final intent=%+v", got)
	}
	if len(events) != 2 {
		t.Fatalf("events=%d", len(events))
	}
}

func TestRefundWebhookRejectsOverdraftAndZeroMeansRemaining(t *testing.T) {
	ctx := context.Background()
	runtime, gateway, _, _, _, _ := newTestRuntime(t)
	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{Product: "test-product", BusinessOrderID: "biz-1", Provider: ProviderXunhu})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := runtime.StartCheckout(ctx, intent.ID); err != nil {
		t.Fatal(err)
	}
	gateway.event = checkoutEvent(intent)
	if err := runtime.HandleWebhookEvent(ctx, []byte("paid"), nil); err != nil {
		t.Fatal(err)
	}

	refund := checkoutEvent(intent)
	refund.Type = EventPaymentRefunded
	refund.Data.PaymentID = "refund-partial"
	refund.Data.Amount = 400
	gateway.event = refund
	if err := runtime.HandleWebhookEvent(ctx, []byte("partial"), nil); err != nil {
		t.Fatal(err)
	}

	// A second refund exceeding the remaining balance must be rejected and
	// must leave the intent untouched (cumulative overdraft guard).
	refund.Data.PaymentID = "refund-overdraft"
	refund.Data.Amount = 700
	gateway.event = refund
	if err := runtime.HandleWebhookEvent(ctx, []byte("overdraft"), nil); err == nil {
		t.Fatal("overdraft refund webhook: want error, got nil")
	}
	got, err := runtime.GetIntent(ctx, intent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != IntentPaid || got.RefundedAmountMinor != 400 {
		t.Fatalf("after overdraft intent=%+v", got)
	}

	// A zero amount means "refund the remaining balance", not the original
	// total — it must complete the refund without overshooting.
	refund.Data.PaymentID = "refund-rest"
	refund.Data.Amount = 0
	gateway.event = refund
	if err := runtime.HandleWebhookEvent(ctx, []byte("rest"), nil); err != nil {
		t.Fatalf("remaining refund webhook: %+v", err)
	}
	got, err = runtime.GetIntent(ctx, intent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != PaymentStatusRefunded || got.RefundedAmountMinor != got.AmountMinor {
		t.Fatalf("final intent=%+v", got)
	}
}

func TestNotificationQueueProcessesEntitlementCompensation(t *testing.T) {
	ctx := context.Background()
	runtime, _, _, sender, _, notifications := newTestRuntime(t)
	if err := notifications.EnqueueOnce(ctx, "entitlement:test",
		NotificationEntitlementCompensation, EmailMessage{
			To: "user@example.com", Subject: "Compensation", Body: "<p>7 days</p>",
		}); err != nil {
		t.Fatal(err)
	}
	if err := runtime.ProcessPendingNotifications(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if len(sender.messages) != 1 || sender.messages[0].Subject != "Compensation" {
		t.Fatalf("messages=%+v", sender.messages)
	}
	pending, err := notifications.Pending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending=%+v", pending)
	}
}
