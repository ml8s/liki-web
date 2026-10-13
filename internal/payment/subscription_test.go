package payment

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

type subscriptionHandlerStub struct {
	events []WebhookEvent
	err    error
}

func (s *subscriptionHandlerStub) HandleSubscriptionEvent(_ context.Context, event WebhookEvent) error {
	if s.err != nil {
		return s.err
	}
	s.events = append(s.events, event)
	return nil
}

func TestRuntimeRoutesSubscriptionEventsBeforeIntentLookup(t *testing.T) {
	handler := &subscriptionHandlerStub{}
	gateway := &fakeGateway{}
	runtime := NewRuntime(
		map[Provider]Gateway{ProviderDodo: gateway}, nil, nil, nil, nil, nil, nil, RuntimeOptions{},
	)
	runtime.SetSubscriptionHandler(handler)

	start := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	event := WebhookEvent{
		Provider: ProviderDodo,
		Type:     EventSubscriptionRenewed,
		Data: WebhookEventData{
			SubscriptionID: "sub_1", Status: "active", ProviderPeriodCount: 2,
			PeriodStart: &start, PeriodEnd: &start,
		},
	}
	gateway.event = event
	ack, err := runtime.HandleWebhookEventWithAck(context.Background(), []byte("body"), http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	if string(ack) != "ok" {
		t.Fatalf("ack = %q", ack)
	}
	if len(handler.events) != 1 || handler.events[0].Type != EventSubscriptionRenewed {
		t.Fatalf("handler events = %+v", handler.events)
	}
}

func TestRuntimeSubscriptionFailureIsRetryable(t *testing.T) {
	gateway := &fakeGateway{}
	runtime := NewRuntime(
		map[Provider]Gateway{ProviderDodo: gateway}, nil, nil, nil, nil, nil, nil, RuntimeOptions{},
	)
	runtime.SetSubscriptionHandler(&subscriptionHandlerStub{err: errors.New("db unavailable")})
	gateway.event = WebhookEvent{Provider: ProviderDodo, Type: EventSubscriptionActive}
	if _, err := runtime.HandleWebhookEventWithAck(context.Background(), []byte("body"), http.Header{}); err == nil {
		t.Fatal("subscription handler error = nil, want retryable error")
	}
}
