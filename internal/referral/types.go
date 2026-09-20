// Package referral owns the two-level referral domain: partner registration,
// 30-day click attribution, and commissions for paid dodo/xunhu intents.
package referral

import (
	"time"
)

type PartnerStatus string

const (
	PartnerActive PartnerStatus = "active"
	PartnerFrozen PartnerStatus = "frozen"
)

type PartnerRole string

const (
	RoleKOL            PartnerRole = "kol"
	RoleChannelPartner PartnerRole = "channel_partner"
)

type CommissionStatus string

const (
	CommissionPending CommissionStatus = "pending"
	CommissionPaid    CommissionStatus = "paid"
	CommissionVoided  CommissionStatus = "voided"
)

type PayoutStatus string

const (
	PayoutPending PayoutStatus = "pending"
	PayoutPaid    PayoutStatus = "paid"
	PayoutFailed  PayoutStatus = "failed"
)

const (
	ReferrerRateBPS       = 5000 // 50%
	RecruiterRateBPS      = 2000 // 20%
	AttributionCookieName = "liki_ref"
	AttributionCookieTTL  = 30 * 24 * time.Hour
)

type Referrer struct {
	UserID      string        `json:"user_id"`
	Code        string        `json:"code"`
	Role        PartnerRole   `json:"role"`
	Status      PartnerStatus `json:"status"`
	RecruiterID string        `json:"recruiter_id,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type Attribution struct {
	UserID     string    `json:"user_id"`
	ReferrerID string    `json:"referrer_id"`
	Code       string    `json:"code"`
	ReferredAt time.Time `json:"referred_at"`
}

type Commission struct {
	ID               string           `json:"id"`
	PaymentProvider  string           `json:"payment_provider"`
	PaymentIntentID  string           `json:"payment_intent_id"`
	BusinessOrderID  string           `json:"business_order_id"`
	Product          string           `json:"product"`
	BuyerUserID      string           `json:"buyer_user_id"`
	ReferrerID       string           `json:"referrer_id"`
	RecruiterID      string           `json:"recruiter_id,omitempty"`
	AmountMinor      int64            `json:"amount_minor"`
	Currency         string           `json:"currency"`
	ReferrerRateBPS  int              `json:"referrer_rate_bps"`
	ReferrerAmount   int64            `json:"referrer_commission_minor"`
	RecruiterRateBPS int              `json:"recruiter_rate_bps,omitempty"`
	RecruiterAmount  int64            `json:"recruiter_commission_minor,omitempty"`
	Status           CommissionStatus `json:"status"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	PaidAt           *time.Time       `json:"paid_at,omitempty"`
}

type ApplicationStatus string

const (
	ApplicationPending  ApplicationStatus = "pending"
	ApplicationApproved ApplicationStatus = "approved"
	ApplicationRejected ApplicationStatus = "rejected"
)

type PartnerApplication struct {
	ID             string            `json:"id"`
	UserID         string            `json:"user_id"`
	Code           string            `json:"code"`
	Status         ApplicationStatus `json:"status"`
	Note           string            `json:"note"`
	ReviewerID     string            `json:"reviewer_id,omitempty"`
	DecisionReason string            `json:"decision_reason,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	ReviewedAt     *time.Time        `json:"reviewed_at,omitempty"`
}

type PartnerAdminAction struct {
	ID         string    `json:"id"`
	Actor      string    `json:"actor"`
	Action     string    `json:"action"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id"`
	Before     string    `json:"before,omitempty"`
	After      string    `json:"after,omitempty"`
	Reason     string    `json:"reason,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type PartnerStats struct {
	Referrer         Referrer     `json:"referrer"`
	AttributedUsers  int          `json:"attributed_users"`
	Commissions      []Commission `json:"commissions"`
	TotalEarnedMinor int64        `json:"total_earned_minor"`
	PendingMinor     int64        `json:"pending_minor"`
	PaidMinor        int64        `json:"paid_minor"`
}
