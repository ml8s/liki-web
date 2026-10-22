package partner

import (
	"errors"
	"testing"
)

func TestDecidePayoutStatus(t *testing.T) {
	if got, err := decidePayoutStatus(PayoutRequested, true); err != nil || got != PayoutApproved {
		t.Fatalf("approve = %v, %v", got, err)
	}
	if got, err := decidePayoutStatus(PayoutRequested, false); err != nil || got != PayoutRejected {
		t.Fatalf("reject = %v, %v", got, err)
	}
	if _, err := decidePayoutStatus(PayoutApproved, true); !errors.Is(err, ErrPayoutInvalidTransition) {
		t.Fatalf("approve from approved = %v", err)
	}
}

func TestMarkPayoutPaidStatus(t *testing.T) {
	if got, err := markPayoutPaidStatus(PayoutApproved); err != nil || got != PayoutPaid {
		t.Fatalf("paid = %v, %v", got, err)
	}
	if _, err := markPayoutPaidStatus(PayoutRequested); !errors.Is(err, ErrPayoutInvalidTransition) {
		t.Fatalf("paid from requested = %v", err)
	}
}

func TestPayoutInputValidation(t *testing.T) {
	if err := (PayoutDecision{}).validate(); err == nil {
		t.Fatal("empty decision must fail")
	}
	if err := (PayoutPayment{Actor: "admin"}).validate(); err == nil {
		t.Fatal("payment without payout/reference must fail")
	}
}
