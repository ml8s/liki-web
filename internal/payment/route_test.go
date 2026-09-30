package payment

import (
	"context"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestCurrencyRoutePolicy(t *testing.T) {
	t.Parallel()
	policy := CurrencyRoutePolicy{}
	available := []Provider{ProviderDodo, ProviderXunhu}

	got, err := policy.Select("CNY", available)
	if err != nil || got != ProviderXunhu {
		t.Fatalf("CNY route = %v, %v; want xunhu, nil", got, err)
	}
	got, err = policy.Select("USD", available)
	if err != nil || got != ProviderDodo {
		t.Fatalf("USD route = %v, %v; want dodo, nil", got, err)
	}
	if _, err := policy.Select("CNY", []Provider{ProviderDodo}); err == nil || !strings.Contains(err.Error(), "no gateway") {
		t.Fatalf("missing gateway err = %v", err)
	}
}

func TestRuntimeCreateIntentIgnoresProviderHint(t *testing.T) {
	ctx := context.Background()
	runtime, _, flow, _, _, _ := newTestRuntime(t)

	intent, err := runtime.CreateIntent(ctx, CreateIntentInput{
		Product:         "test-product",
		BusinessOrderID: "biz-1",
		Provider:        ProviderDodo,
	})
	if err != nil {
		t.Fatal(err)
	}
	if intent.Provider != ProviderXunhu {
		t.Fatalf("provider = %q, want xunhu", intent.Provider)
	}
	if flow.order.ID != "biz-1" {
		t.Fatal("flow was not consulted")
	}
}

func TestFulfillmentTaskRetriesWithBackoff(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := newRuntimeTestDB(t)
	tasks := NewFulfillmentTaskStore(db)
	if err := tasks.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	intent := PaymentIntent{
		ID: "pi-retry", BusinessOrderID: "biz-retry", Product: "test-product",
	}
	if err := tasks.EnsureQueued(ctx, intent); err != nil {
		t.Fatal(err)
	}

	task, err := tasks.ClaimNext(ctx, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := tasks.MarkFailed(ctx, task.ID, context.DeadlineExceeded); err != nil {
		t.Fatal(err)
	}

	if _, err := tasks.ClaimNext(ctx, time.Now().UTC()); err != ErrNoRunnableFulfillment {
		t.Fatalf("claim during backoff = %v, want ErrNoRunnableFulfillment", err)
	}
	if _, err := db.Exec(`UPDATE fulfillment_tasks SET run_at=? WHERE id=?`,
		time.Now().UTC().Add(-time.Second).Format(time.RFC3339), task.ID); err != nil {
		t.Fatal(err)
	}

	task, err = tasks.ClaimNext(ctx, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if task.Attempts != 2 {
		t.Fatalf("attempts = %d, want 2", task.Attempts)
	}
}
