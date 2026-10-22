package partner

import (
	"errors"
	"testing"
)

func TestPayoutStatusTransitions(t *testing.T) {
	if got, err := decidePayoutStatus(PayoutRequested, true); err != nil || got != PayoutApproved {
		t.Fatalf("approve = %v, %v", got, err)
	}
	if got, err := decidePayoutStatus(PayoutRequested, false); err != nil || got != PayoutRejected {
		t.Fatalf("reject = %v, %v", got, err)
	}
	if _, err := decidePayoutStatus(PayoutApproved, true); !errors.Is(err, ErrPayoutInvalidTransition) {
		t.Fatalf("approve from approved = %v", err)
	}
	if got, err := markPayoutPaidStatus(PayoutApproved); err != nil || got != PayoutPaid {
		t.Fatalf("paid = %v, %v", got, err)
	}
	if _, err := markPayoutPaidStatus(PayoutRequested); !errors.Is(err, ErrPayoutInvalidTransition) {
		t.Fatalf("paid from requested = %v", err)
	}
}

func TestPayoutInputValidation(t *testing.T) {
	if err := (PayoutDecision{}).validate(); err == nil {
		t.Fatal("empty rejection must fail")
	}
	if err := (PayoutApproval{}).validate(); err == nil {
		t.Fatal("empty approval must fail")
	}
	if err := (PayoutPayment{Actor: "admin"}).validate(); err == nil {
		t.Fatal("payment without payout/reference must fail")
	}
}
