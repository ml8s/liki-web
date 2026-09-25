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
	RoleKOL   PartnerRole = "kol"
	RoleScout PartnerRole = "scout"
)

// PartnerRole is also the application type. Keeping one vocabulary avoids a
// separate, drifting "application type" model.

type CommissionStatus string

const (
	CommissionPending CommissionStatus = "pending"
	CommissionPaid    CommissionStatus = "paid"
	CommissionVoided  CommissionStatus = "voided"
)

type RoleStatus string

const (
	RoleActive  RoleStatus = "active"
	RoleFrozen  RoleStatus = "frozen"
	RoleRetired RoleStatus = "retired"
)

type PayoutStatus string

const (
	PayoutRequested PayoutStatus = "requested"
	PayoutPending   PayoutStatus = "pending"
	PayoutApproved  PayoutStatus = "approved"
	PayoutRejected  PayoutStatus = "rejected"
	PayoutPaid      PayoutStatus = "paid"
	PayoutFailed    PayoutStatus = "failed"
)

type PayoutMethod string

const (
	PayoutMethodAlipay PayoutMethod = "alipay"
	PayoutMethodPayPal PayoutMethod = "paypal"
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
	BeneficiaryRole  PartnerRole      `json:"beneficiary_role,omitempty"`
	AmountMinor      int64            `json:"amount_minor"`
	Currency         string           `json:"currency"`
	ReferrerRateBPS  int              `json:"referrer_rate_bps"`
	ReferrerAmount   int64            `json:"referrer_commission_minor"`
	RecruiterRateBPS int              `json:"recruiter_rate_bps,omitempty"`
	RecruiterAmount  int64            `json:"recruiter_commission_minor,omitempty"`
	Status           CommissionStatus `json:"status"`
	RefundedMinor    int64            `json:"refunded_minor"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	PaidAt           *time.Time       `json:"paid_at,omitempty"`
}

// CommissionEntry is one beneficiary's payable share of a sale. KOL and Scout
// shares are independently stateful and can be locked into separate payouts.
type CommissionEntry struct {
	ID              string           `json:"id"`
	CommissionID    string           `json:"commission_id"`
	BeneficiaryID   string           `json:"beneficiary_id"`
	BeneficiaryRole PartnerRole      `json:"beneficiary_role"`
	AmountMinor     int64            `json:"amount_minor"`
	RefundedMinor   int64            `json:"refunded_minor"`
	Currency        string           `json:"currency"`
	Status          CommissionStatus `json:"status"`
	PayoutID        string           `json:"payout_id,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	PaidAt          *time.Time       `json:"paid_at,omitempty"`
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
	Type           PartnerRole       `json:"type"`
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

type FinanceTaskStatus string

const (
	FinanceTaskPending  FinanceTaskStatus = "pending"
	FinanceTaskResolved FinanceTaskStatus = "resolved"
	FinanceTaskRejected FinanceTaskStatus = "rejected"
)

type FinanceTaskType string

const (
	FinanceTaskCommissionClawback FinanceTaskType = "commission_refund_clawback"
	FinanceTaskUserErasure        FinanceTaskType = "user_erasure_review"
)

type FinanceTask struct {
	ID               string            `json:"id"`
	Type             FinanceTaskType   `json:"type"`
	Provider         string            `json:"provider"`
	PaymentIntentID  string            `json:"payment_intent_id"`
	ProviderRefundID string            `json:"provider_refund_id"`
	EntryID          string            `json:"entry_id"`
	BeneficiaryID    string            `json:"beneficiary_id"`
	BeneficiaryRole  PartnerRole       `json:"beneficiary_role"`
	AmountMinor      int64             `json:"amount_minor"`
	Currency         string            `json:"currency"`
	Status           FinanceTaskStatus `json:"status"`
	Assignee         string            `json:"assignee,omitempty"`
	ResolutionNote   string            `json:"resolution_note,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	ResolvedAt       *time.Time        `json:"resolved_at,omitempty"`
}

type Payout struct {
	ID                   string       `json:"id"`
	ReferrerID           string       `json:"referrer_id"`
	Role                 PartnerRole  `json:"role"`
	AmountMinor          int64        `json:"amount_minor"`
	RequestedAmountMinor int64        `json:"requested_amount_minor"`
	Currency             string       `json:"currency"`
	Method               PayoutMethod `json:"method"`
	Account              string       `json:"account"`
	Status               PayoutStatus `json:"status"`
	PaymentReference     string       `json:"payment_reference,omitempty"`
	RejectionReason      string       `json:"rejection_reason,omitempty"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"updated_at"`
	PaidAt               *time.Time   `json:"paid_at,omitempty"`
}

type CurrencyAmount struct {
	Currency    string `json:"currency"`
	AmountMinor int64  `json:"amount_minor"`
}

type BalanceSummary struct {
	Currency     string `json:"currency"`
	TotalMinor   int64  `json:"total_minor"`
	PendingMinor int64  `json:"pending_minor"`
	PaidMinor    int64  `json:"paid_minor"`
}

type ScoutBalanceSummary struct {
	Currency        string `json:"currency"`
	SalesMinor      int64  `json:"sales_minor"`
	CommissionMinor int64  `json:"commission_minor"`
	PendingMinor    int64  `json:"pending_minor"`
	PaidMinor       int64  `json:"paid_minor"`
}

// Settlement is the money-only view available to active and retired role
// capabilities. It deliberately excludes recruiting/dashboard data.
type Settlement struct {
	Role     PartnerRole      `json:"role"`
	Status   RoleStatus       `json:"status"`
	Balances []BalanceSummary `json:"balances"`
	Payouts  []Payout         `json:"payouts"`
}

type KOLOverview struct {
	Referrer        Referrer         `json:"referrer"`
	AttributedUsers int              `json:"attributed_users"`
	Balances        []BalanceSummary `json:"balances"`
}

type ScoutRecruit struct {
	UserID             string           `json:"user_id"`
	Code               string           `json:"code"`
	CreatedAt          time.Time        `json:"created_at"`
	OrderCount         int64            `json:"order_count"`
	CommissionBalances []CurrencyAmount `json:"commission_balances"`
}

type ScoutOverview struct {
	Partner      Referrer              `json:"partner"`
	LeadCount    int64                 `json:"lead_count"`
	RecruitCount int64                 `json:"recruit_count"`
	OrderCount   int64                 `json:"order_count"`
	Balances     []ScoutBalanceSummary `json:"balances"`
	Recruits     []ScoutRecruit        `json:"recruits"`
}

type ScoutSale struct {
	OrderRef             string           `json:"order_ref"`
	Product              string           `json:"product"`
	BuyerRef             string           `json:"buyer_ref"`
	AmountMinor          int64            `json:"amount_minor"`
	Currency             string           `json:"currency"`
	ScoutCommissionMinor int64            `json:"scout_commission_minor"`
	Status               CommissionStatus `json:"status"`
	RefundedMinor        int64            `json:"refunded_minor"`
	CreatedAt            time.Time        `json:"created_at"`
}

type PartnerStats struct {
	Referrer        Referrer         `json:"referrer"`
	AttributedUsers int              `json:"attributed_users"`
	Commissions     []Commission     `json:"commissions"`
	Balances        []BalanceSummary `json:"balances"`
}
