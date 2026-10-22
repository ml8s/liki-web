package partner

import (
	"errors"
	"strings"
)

// PayoutDecision is an explicit rejection fact. Approvals use PayoutApproval
// because they do not populate rejection_reason.
type PayoutDecision struct {
	PayoutID string
	Actor    string
	Reason   string
}

func (decision PayoutDecision) validate() error {
	if strings.TrimSpace(decision.PayoutID) == "" {
		return errors.New("partner: payout id is required")
	}
	if strings.TrimSpace(decision.Actor) == "" {
		return errors.New("partner: actor is required")
	}
	if strings.TrimSpace(decision.Reason) == "" {
		return errors.New("partner: reason is required")
	}
	return nil
}

// PayoutApproval is an explicit approval fact. Its audit reason is canonical.
type PayoutApproval struct {
	PayoutID string
	Actor    string
}

func (approval PayoutApproval) validate() error {
	if strings.TrimSpace(approval.PayoutID) == "" {
		return errors.New("partner: payout id is required")
	}
	if strings.TrimSpace(approval.Actor) == "" {
		return errors.New("partner: actor is required")
	}
	return nil
}

// PayoutPayment is the explicit settlement fact for an approved payout.
type PayoutPayment struct {
	PayoutID         string
	Actor            string
	PaymentReference string
}

func (payment PayoutPayment) validate() error {
	if strings.TrimSpace(payment.PayoutID) == "" {
		return errors.New("partner: payout id is required")
	}
	if strings.TrimSpace(payment.Actor) == "" {
		return errors.New("partner: actor is required")
	}
	if strings.TrimSpace(payment.PaymentReference) == "" {
		return errors.New("partner: payment reference is required")
	}
	return nil
}

// decidePayoutStatus is the single authority for review transitions.
func decidePayoutStatus(current PayoutStatus, approved bool) (PayoutStatus, error) {
	if current != PayoutRequested {
		return "", ErrPayoutInvalidTransition
	}
	if approved {
		return PayoutApproved, nil
	}
	return PayoutRejected, nil
}

// markPayoutPaidStatus is the single authority for settlement transition.
func markPayoutPaidStatus(current PayoutStatus) (PayoutStatus, error) {
	if current != PayoutApproved {
		return "", ErrPayoutInvalidTransition
	}
	return PayoutPaid, nil
}
