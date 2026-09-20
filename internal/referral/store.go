package referral

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"liki-web/internal/migrate"
	"liki-web/internal/sqliteutil"
)

var (
	ErrNotEligible         = errors.New("referral: active liki-chat entitlement required")
	ErrAlreadyReferrer     = errors.New("referral: user is already a referrer")
	ErrInvalidCode         = errors.New("referral: invalid referral code")
	ErrSelfReferral        = errors.New("referral: self referral is not allowed")
	ErrUnsupportedProvider = errors.New("referral: payment provider is not supported")
)

// referralCommissionProduct is the product sold by the referral programme.
// Payments routed through dodo or xunhu are eligible only after the product
// fulfillment succeeds.
const referralCommissionProduct = "liki-chat"

// PaidAccessChecker keeps this package decoupled from liki entitlement storage.
type PaidAccessChecker func(ctx context.Context, userID string) (bool, error)

type Store struct {
	db        *sql.DB
	nowFunc   func() time.Time
	idFunc    func() string
	codeFunc  func() (string, error)
	paidCheck PaidAccessChecker
}

func Open(path string, paidCheck PaidAccessChecker) (*Store, error) {
	if paidCheck == nil {
		return nil, errors.New("referral: paid access checker is required")
	}
	db, err := sqliteutil.Open(path)
	if err != nil {
		return nil, err
	}
	return &Store{
		db: db, paidCheck: paidCheck,
		nowFunc:  func() time.Time { return time.Now().UTC() },
		idFunc:   uuid.NewString,
		codeFunc: newPartnerCode,
	}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Migrate(ctx context.Context) error {
	return migrate.Run(ctx, s.db, []migrate.Migration{{
		ID: "001_referral_core",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS referrers (
    user_id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    role TEXT NOT NULL CHECK (role IN ('kol')),
    status TEXT NOT NULL CHECK (status IN ('active','frozen')),
    recruiter_id TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_referrers_recruiter
    ON referrers(recruiter_id, status);

CREATE TABLE IF NOT EXISTS referral_attributions (
    user_id TEXT PRIMARY KEY,
    referrer_id TEXT NOT NULL,
    code TEXT NOT NULL,
    referred_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_referral_attributions_referrer
    ON referral_attributions(referrer_id, referred_at);

CREATE TABLE IF NOT EXISTS referral_commissions (
    id TEXT PRIMARY KEY,
    payment_provider TEXT NOT NULL CHECK (payment_provider IN ('dodo','xunhu')),
    payment_intent_id TEXT NOT NULL,
    business_order_id TEXT NOT NULL,
    product TEXT NOT NULL,
    buyer_user_id TEXT NOT NULL,
    referrer_id TEXT NOT NULL,
    recruiter_id TEXT,
    amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
    currency TEXT NOT NULL,
    referrer_rate_bps INTEGER NOT NULL CHECK (referrer_rate_bps BETWEEN 0 AND 10000),
    referrer_commission_minor INTEGER NOT NULL CHECK (referrer_commission_minor >= 0),
    recruiter_rate_bps INTEGER NOT NULL DEFAULT 0 CHECK (recruiter_rate_bps BETWEEN 0 AND 10000),
    recruiter_commission_minor INTEGER NOT NULL DEFAULT 0 CHECK (recruiter_commission_minor >= 0),
    status TEXT NOT NULL CHECK (status IN ('pending','paid','voided')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    paid_at TEXT,
    UNIQUE(payment_provider, payment_intent_id)
);
CREATE INDEX IF NOT EXISTS idx_referral_commissions_referrer
    ON referral_commissions(referrer_id, status, created_at);

CREATE TABLE IF NOT EXISTS referral_payouts (
    id TEXT PRIMARY KEY,
    referrer_id TEXT NOT NULL,
    amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
    currency TEXT NOT NULL,
    method TEXT NOT NULL CHECK (method IN ('alipay','paypal')),
    account TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending','paid','failed')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    paid_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_referral_payouts_referrer
    ON referral_payouts(referrer_id, status, created_at);
`)
			return err
		},
	}, {
		ID: "003_partner_admin",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
CREATE TABLE referrers_admin_migrate (
    user_id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    role TEXT NOT NULL CHECK (role IN ('kol','channel_partner')),
    status TEXT NOT NULL CHECK (status IN ('active','frozen')),
    recruiter_id TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
INSERT INTO referrers_admin_migrate
    (user_id,code,role,status,recruiter_id,created_at,updated_at)
SELECT user_id,code,role,status,recruiter_id,created_at,updated_at FROM referrers;
DROP TABLE referrers;
ALTER TABLE referrers_admin_migrate RENAME TO referrers;
CREATE INDEX IF NOT EXISTS idx_referrers_recruiter
    ON referrers(recruiter_id,status);

CREATE TABLE IF NOT EXISTS referral_applications (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type='channel_partner'),
    status TEXT NOT NULL CHECK (status IN ('pending','approved','rejected')),
    note TEXT NOT NULL DEFAULT '',
    reviewer_id TEXT,
    decision_reason TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    reviewed_at TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_referral_applications_pending_user
    ON referral_applications(user_id) WHERE status='pending';
CREATE INDEX IF NOT EXISTS idx_referral_applications_status
    ON referral_applications(status,created_at);

CREATE TABLE IF NOT EXISTS referral_admin_actions (
    id TEXT PRIMARY KEY,
    actor TEXT NOT NULL,
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    before TEXT,
    after TEXT,
    reason TEXT,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_referral_admin_actions_created
    ON referral_admin_actions(created_at);
`)
			return err
		},
	}, {
		ID: "002_commission_intake",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS commission_intents (
    id TEXT PRIMARY KEY,
    payment_provider TEXT NOT NULL,
    payment_intent_id TEXT NOT NULL,
    business_order_id TEXT NOT NULL,
    product TEXT NOT NULL,
    buyer_user_id TEXT NOT NULL,
    amount_minor INTEGER NOT NULL,
    currency TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued','running','succeeded','failed','skipped')),
    skip_reason TEXT,
    attempts INTEGER NOT NULL DEFAULT 0,
    run_at TEXT NOT NULL,
    last_error TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(payment_provider,payment_intent_id)
);
CREATE INDEX IF NOT EXISTS idx_commission_intents_due
    ON commission_intents(status,run_at);
`)
			return err
		},
	}})
}

func newPartnerCode() (string, error) {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	code := strings.ToUpper(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw))
	code = strings.Map(func(r rune) rune {
		switch r {
		case 'O':
			return 'Q'
		case 'I':
			return 'J'
		case '0':
			return 'M'
		case '1':
			return 'N'
		}
		return r
	}, code)
	return code[:10], nil
}

func formatTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func parseTime(v string) time.Time {
	t, _ := time.Parse(time.RFC3339, v)
	return t
}

func scanReferrer(scan func(dest ...any) error) (Referrer, error) {
	var item Referrer
	var role, status, createdAt, updatedAt string
	var recruiter sql.NullString
	if err := scan(&item.UserID, &item.Code, &role, &status, &recruiter, &createdAt, &updatedAt); err != nil {
		return Referrer{}, err
	}
	item.Role = PartnerRole(role)
	item.Status = PartnerStatus(status)
	item.RecruiterID = recruiter.String
	item.CreatedAt = parseTime(createdAt)
	item.UpdatedAt = parseTime(updatedAt)
	return item, nil
}

func (s *Store) RegisterReferrer(ctx context.Context, userID string) (Referrer, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return Referrer{}, ErrNotEligible
	}
	eligible, err := s.paidCheck(ctx, userID)
	if err != nil {
		return Referrer{}, fmt.Errorf("referral: check paid access: %w", err)
	}
	if !eligible {
		return Referrer{}, ErrNotEligible
	}

	// The recruiter relationship is derived from the referral link that created
	// this account. Users never type a promotion or recruiter code.
	var recruiterID string
	if attribution, attributed, err := s.AttributionForUser(ctx, userID); err != nil {
		return Referrer{}, err
	} else if attributed && attribution.ReferrerID != userID {
		recruiterID = attribution.ReferrerID
	}

	now := formatTime(s.nowFunc())
	for range 5 {
		code, err := s.codeFunc()
		if err != nil {
			return Referrer{}, err
		}
		_, err = s.db.ExecContext(ctx, `
INSERT INTO referrers (user_id,code,role,status,recruiter_id,created_at,updated_at)
VALUES (?,?,'kol','active',?,?,?)
ON CONFLICT(user_id) DO NOTHING`,
			userID, code, recruiterID, now, now)
		if err != nil {
			return Referrer{}, err
		}
		item, err := s.ReferrerByUser(ctx, userID)
		if err == nil {
			return item, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return Referrer{}, err
		}
	}
	return Referrer{}, errors.New("referral: could not allocate unique code")
}

func (s *Store) ReferrerByUser(ctx context.Context, userID string) (Referrer, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT user_id,code,role,status,recruiter_id,created_at,updated_at
FROM referrers WHERE user_id=?`, userID)
	return scanReferrer(row.Scan)
}

func (s *Store) ReferrerByCode(ctx context.Context, code string) (Referrer, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT user_id,code,role,status,recruiter_id,created_at,updated_at
FROM referrers WHERE code=? AND status='active'`, strings.ToUpper(strings.TrimSpace(code)))
	return scanReferrer(row.Scan)
}

func (s *Store) Attribute(ctx context.Context, userID, code string) (Attribution, error) {
	userID = strings.TrimSpace(userID)
	code = strings.ToUpper(strings.TrimSpace(code))
	if userID == "" || code == "" {
		return Attribution{}, ErrInvalidCode
	}
	referrer, err := s.ReferrerByCode(ctx, code)
	if err != nil {
		return Attribution{}, err
	}
	if referrer.UserID == userID {
		return Attribution{}, ErrSelfReferral
	}
	if referrer.Status != PartnerActive {
		return Attribution{}, ErrInvalidCode
	}
	now := formatTime(s.nowFunc())
	_, err = s.db.ExecContext(ctx, `
INSERT INTO referral_attributions (user_id,referrer_id,code,referred_at)
VALUES (?,?,?,?)
ON CONFLICT(user_id) DO NOTHING`,
		userID, referrer.UserID, referrer.Code, now)
	if err != nil {
		return Attribution{}, err
	}
	var item Attribution
	var referred string
	err = s.db.QueryRowContext(ctx, `
SELECT user_id,referrer_id,code,referred_at
FROM referral_attributions WHERE user_id=?`, userID).Scan(
		&item.UserID, &item.ReferrerID, &item.Code, &referred)
	item.ReferredAt = parseTime(referred)
	return item, err
}

func (s *Store) AttributionForUser(ctx context.Context, userID string) (Attribution, bool, error) {
	var item Attribution
	var referred string
	err := s.db.QueryRowContext(ctx, `
SELECT user_id,referrer_id,code,referred_at
FROM referral_attributions WHERE user_id=?`, userID).Scan(
		&item.UserID, &item.ReferrerID, &item.Code, &referred)
	if errors.Is(err, sql.ErrNoRows) {
		return Attribution{}, false, nil
	}
	if err != nil {
		return Attribution{}, false, err
	}
	item.ReferredAt = parseTime(referred)
	return item, true, nil
}

func commission_parts(amount int64, rate int) int64 {
	if amount <= 0 || rate <= 0 {
		return 0
	}
	if amount > (1<<62)/int64(rate) {
		return 0
	}
	return amount * int64(rate) / 10000
}

// RecordCommission records one dodo/xunhu paid intent exactly once. Missing
// attribution is a normal no-op, while a conflict with different data is an
// error so retry can surface corruption instead of hiding it.
func (s *Store) commission(ctx context.Context, provider, intentID string) (Commission, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id,payment_provider,payment_intent_id,business_order_id,product,buyer_user_id,
       referrer_id,recruiter_id,amount_minor,currency,referrer_rate_bps,
       referrer_commission_minor,recruiter_rate_bps,recruiter_commission_minor,
       status,created_at,updated_at,paid_at
FROM referral_commissions
WHERE payment_provider=? AND payment_intent_id=?`, provider, intentID)
	return scanCommission(row.Scan)
}

func scanCommission(scan func(dest ...any) error) (Commission, error) {
	var item Commission
	var provider, status, currency, createdAt, updatedAt string
	var recruiter sql.NullString
	var paid sql.NullString
	if err := scan(
		&item.ID, &provider, &item.PaymentIntentID, &item.BusinessOrderID, &item.Product,
		&item.BuyerUserID, &item.ReferrerID, &recruiter, &item.AmountMinor, &currency,
		&item.ReferrerRateBPS, &item.ReferrerAmount, &item.RecruiterRateBPS,
		&item.RecruiterAmount, &status, &createdAt, &updatedAt, &paid,
	); err != nil {
		return Commission{}, err
	}
	item.PaymentProvider = provider
	item.RecruiterID = recruiter.String
	item.Currency = currency
	item.Status = CommissionStatus(status)
	item.CreatedAt = parseTime(createdAt)
	item.UpdatedAt = parseTime(updatedAt)
	if paid.Valid {
		item.PaidAt = func() *time.Time { t := parseTime(paid.String); return &t }()
	}
	return item, nil
}

func (s *Store) ListCommissions(ctx context.Context, referrerID string, limit int) ([]Commission, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id,payment_provider,payment_intent_id,business_order_id,product,buyer_user_id,
       referrer_id,recruiter_id,amount_minor,currency,referrer_rate_bps,
       referrer_commission_minor,recruiter_rate_bps,recruiter_commission_minor,
       status,created_at,updated_at,paid_at
FROM referral_commissions
WHERE referrer_id=? OR recruiter_id=?
ORDER BY created_at DESC LIMIT ?`, referrerID, referrerID, limit)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]Commission, 0)
	for rows.Next() {
		item, scanErr := scanCommission(rows.Scan)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var earned int64
	if err := s.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(referrer_commission_minor),0)
FROM referral_commissions WHERE referrer_id=?`, referrerID).Scan(&earned); err != nil {
		return nil, 0, err
	}
	return items, earned, nil
}

func (s *Store) CountAttributed(ctx context.Context, referrerID string) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `
SELECT count(*) FROM referral_attributions WHERE referrer_id=?`, referrerID).Scan(&n)
	return n, err
}

func (s *Store) PartnerStats(ctx context.Context, referrerID string) (PartnerStats, error) {
	item, err := s.ReferrerByUser(ctx, referrerID)
	if err != nil {
		return PartnerStats{}, err
	}
	commissions, earned, err := s.ListCommissions(ctx, referrerID, 100)
	if err != nil {
		return PartnerStats{}, err
	}
	attributed, err := s.CountAttributed(ctx, referrerID)
	if err != nil {
		return PartnerStats{}, err
	}
	var pending int64
	if err := s.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(referrer_commission_minor),0)
FROM referral_commissions WHERE referrer_id=? AND status='pending'`, referrerID).Scan(&pending); err != nil {
		return PartnerStats{}, err
	}
	var paid int64
	if err := s.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(referrer_commission_minor),0)
FROM referral_commissions WHERE referrer_id=? AND status='paid'`, referrerID).Scan(&paid); err != nil {
		return PartnerStats{}, err
	}
	return PartnerStats{
		Referrer: item, AttributedUsers: int(attributed), Commissions: commissions,
		TotalEarnedMinor: earned, PendingMinor: pending, PaidMinor: paid,
	}, nil
}
