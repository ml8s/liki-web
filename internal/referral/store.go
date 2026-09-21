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
	accounts  *fieldSealer
}

func Open(path string, payoutSecret []byte, paidCheck PaidAccessChecker) (*Store, error) {
	if paidCheck == nil {
		return nil, errors.New("referral: paid access checker is required")
	}
	accounts, err := newFieldSealer(payoutSecret)
	if err != nil {
		return nil, err
	}
	db, err := sqliteutil.Open(path)
	if err != nil {
		return nil, err
	}
	return &Store{
		db: db, paidCheck: paidCheck, accounts: accounts,
		nowFunc:  func() time.Time { return time.Now().UTC() },
		idFunc:   uuid.NewString,
		codeFunc: newPartnerCode,
	}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Migrate(ctx context.Context) error {
	return migrate.Run(ctx, s.db, s.migrations())
}

func (s *Store) migrations() []migrate.Migration {
	return []migrate.Migration{{
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
	}, {
		ID: "003_partner_admin",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
CREATE TABLE referrers_admin_migrate (
    user_id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    role TEXT NOT NULL CHECK (role IN ('kol','scout')),
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
    type TEXT NOT NULL CHECK (type='scout'),
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
		ID: "004_payout_state",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
ALTER TABLE referral_commissions ADD COLUMN payout_id TEXT;
CREATE INDEX IF NOT EXISTS idx_referral_commissions_payout
    ON referral_commissions(payout_id);
DROP INDEX IF EXISTS idx_referral_payouts_referrer;
ALTER TABLE referral_payouts RENAME TO referral_payouts_admin_migrate;
CREATE TABLE referral_payouts (
    id TEXT PRIMARY KEY,
    referrer_id TEXT NOT NULL,
    amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
    currency TEXT NOT NULL,
    method TEXT NOT NULL CHECK (method IN ('alipay','paypal')),
    account TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('requested','pending','approved','rejected','paid','failed')),
    payment_reference TEXT,
    rejection_reason TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    paid_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_referral_payouts_referrer
    ON referral_payouts(referrer_id,status,created_at);
CREATE INDEX IF NOT EXISTS idx_referral_payouts_status
    ON referral_payouts(status,created_at);
INSERT INTO referral_payouts
    (id,referrer_id,amount_minor,currency,method,account,status,
     created_at,updated_at,paid_at)
SELECT id,referrer_id,amount_minor,currency,method,account,status,
       created_at,updated_at,paid_at
FROM referral_payouts_admin_migrate;
DROP TABLE referral_payouts_admin_migrate;
`)
			return err
		}}, {
		ID: "005_scout_role",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
CREATE TABLE referrers_scout_migrate (
    user_id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    role TEXT NOT NULL CHECK (role IN ('kol','scout')),
    status TEXT NOT NULL CHECK (status IN ('active','frozen')),
    recruiter_id TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
INSERT INTO referrers_scout_migrate
    (user_id,code,role,status,recruiter_id,created_at,updated_at)
SELECT user_id,code,
       CASE role WHEN 'channel_partner' THEN 'scout' ELSE role END,
       status,recruiter_id,created_at,updated_at
FROM referrers;
DROP INDEX IF EXISTS idx_referrers_recruiter;
DROP TABLE referrers;
ALTER TABLE referrers_scout_migrate RENAME TO referrers;
CREATE INDEX IF NOT EXISTS idx_referrers_recruiter
    ON referrers(recruiter_id,status);

ALTER TABLE referral_applications RENAME TO referral_applications_scout_migrate;
CREATE TABLE referral_applications (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type='scout'),
    status TEXT NOT NULL CHECK (status IN ('pending','approved','rejected')),
    note TEXT NOT NULL DEFAULT '',
    reviewer_id TEXT,
    decision_reason TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    reviewed_at TEXT
);
INSERT INTO referral_applications
    (id,user_id,type,status,note,reviewer_id,decision_reason,created_at,updated_at,reviewed_at)
SELECT id,user_id,'scout',status,note,reviewer_id,decision_reason,created_at,updated_at,reviewed_at
FROM referral_applications_scout_migrate;
DROP INDEX IF EXISTS idx_referral_applications_pending_user;
DROP INDEX IF EXISTS idx_referral_applications_status;
DROP TABLE referral_applications_scout_migrate;
CREATE UNIQUE INDEX IF NOT EXISTS idx_referral_applications_pending_user
    ON referral_applications(user_id) WHERE status='pending';
CREATE INDEX IF NOT EXISTS idx_referral_applications_status
    ON referral_applications(status,created_at);
`)
			return err
		},
	}, {
		ID: "006_commission_entries",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
ALTER TABLE referral_commissions ADD COLUMN refunded_minor INTEGER NOT NULL DEFAULT 0;

CREATE TABLE referral_commission_entries (
    id TEXT PRIMARY KEY,
    commission_id TEXT NOT NULL,
    beneficiary_id TEXT NOT NULL,
    beneficiary_role TEXT NOT NULL CHECK (beneficiary_role IN ('kol','scout')),
    amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
    refunded_minor INTEGER NOT NULL DEFAULT 0 CHECK (
        refunded_minor >= 0 AND refunded_minor <= amount_minor
    ),
    currency TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending','paid','voided')),
    payout_id TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    paid_at TEXT,
    UNIQUE (commission_id, beneficiary_role)
);
CREATE INDEX idx_commission_entries_beneficiary
    ON referral_commission_entries(beneficiary_id, status, created_at);
CREATE INDEX idx_commission_entries_payout
    ON referral_commission_entries(payout_id);

INSERT INTO referral_commission_entries
    (id,commission_id,beneficiary_id,beneficiary_role,amount_minor,refunded_minor,
     currency,status,payout_id,created_at,updated_at,paid_at)
SELECT c.id||':kol',c.id,c.referrer_id,'kol',c.referrer_commission_minor,0,
       c.currency,
       CASE
         WHEN p.id IS NULL OR pr.role='kol' THEN c.status
         WHEN c.status='voided' THEN 'voided'
         ELSE 'pending'
       END,
       CASE WHEN p.id IS NOT NULL AND pr.role='kol' THEN c.payout_id END,
       c.created_at,c.updated_at,
       CASE WHEN p.id IS NOT NULL AND pr.role='kol' THEN c.paid_at END
FROM referral_commissions c
LEFT JOIN referral_payouts p ON p.id=c.payout_id
LEFT JOIN referrers pr ON pr.user_id=p.referrer_id
WHERE c.referrer_id != '' AND c.referrer_commission_minor > 0;

INSERT INTO referral_commission_entries
    (id,commission_id,beneficiary_id,beneficiary_role,amount_minor,refunded_minor,
     currency,status,payout_id,created_at,updated_at,paid_at)
SELECT c.id||':scout',c.id,c.recruiter_id,'scout',c.recruiter_commission_minor,0,
       c.currency,
       CASE
         WHEN p.id IS NULL OR pr.role='scout' THEN c.status
         WHEN c.status='voided' THEN 'voided'
         ELSE 'pending'
       END,
       CASE WHEN p.id IS NOT NULL AND pr.role='scout' THEN c.payout_id END,
       c.created_at,c.updated_at,
       CASE WHEN p.id IS NOT NULL AND pr.role='scout' THEN c.paid_at END
FROM referral_commissions c
LEFT JOIN referral_payouts p ON p.id=c.payout_id
LEFT JOIN referrers pr ON pr.user_id=p.referrer_id
WHERE c.recruiter_id IS NOT NULL AND c.recruiter_id != '' AND c.recruiter_commission_minor > 0;
`)
			return err
		},
	}, {
		ID: "007_commission_refund_intake",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
CREATE TABLE commission_refund_intents (
    id TEXT PRIMARY KEY,
    payment_provider TEXT NOT NULL,
    payment_intent_id TEXT NOT NULL,
    provider_refund_id TEXT NOT NULL,
    amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
    currency TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued','running','succeeded','skipped')),
    skip_reason TEXT,
    attempts INTEGER NOT NULL DEFAULT 0,
    run_at TEXT NOT NULL,
    last_error TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (payment_provider, provider_refund_id)
);
CREATE INDEX idx_commission_refund_intents_due
    ON commission_refund_intents(status,run_at);
`)
			return err
		},
	}, {
		ID: "008_partner_roles",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
CREATE TABLE partner_roles (
    user_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('kol','scout')),
    status TEXT NOT NULL CHECK (status IN ('active','frozen','retired')),
    started_at TEXT NOT NULL,
    ended_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (user_id, role)
);
CREATE INDEX idx_partner_roles_role_status
    ON partner_roles(role,status,started_at);

INSERT INTO partner_roles
    (user_id,role,status,started_at,created_at,updated_at)
SELECT user_id,role,status,created_at,updated_at,updated_at
FROM referrers;

-- Promotion must not orphan historical KOL earnings. Preserve retired KOL
-- capability when that beneficiary still owns KOL entries; retired role may
-- settle money but cannot access the KOL console.
INSERT INTO partner_roles
    (user_id,role,status,started_at,created_at,updated_at)
SELECT DISTINCT e.beneficiary_id,'kol','retired',e.created_at,
       strftime('%Y-%m-%dT%H:%M:%SZ','now'),strftime('%Y-%m-%dT%H:%M:%SZ','now')
FROM referral_commission_entries e
JOIN referrers r ON r.user_id=e.beneficiary_id
WHERE e.beneficiary_role='kol' AND r.role='scout'
  AND e.status IN ('pending','paid')
ON CONFLICT(user_id,role) DO NOTHING;
`)
			return err
		},
	}, {
		ID: "009_financial_ledger",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, `
ALTER TABLE commission_refund_intents RENAME TO commission_refund_intents_missing_failed;
DROP INDEX IF EXISTS idx_commission_refund_intents_due;
CREATE TABLE commission_refund_intents (
    id TEXT PRIMARY KEY,
    payment_provider TEXT NOT NULL,
    payment_intent_id TEXT NOT NULL,
    provider_refund_id TEXT NOT NULL,
    amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
    currency TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued','running','succeeded','failed','skipped')),
    skip_reason TEXT,
    attempts INTEGER NOT NULL DEFAULT 0,
    run_at TEXT NOT NULL,
    last_error TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (payment_provider, provider_refund_id)
);
CREATE INDEX idx_commission_refund_intents_due
    ON commission_refund_intents(status,run_at);

INSERT INTO commission_refund_intents
    (id,payment_provider,payment_intent_id,provider_refund_id,amount_minor,currency,
     status,skip_reason,attempts,run_at,last_error,created_at,updated_at)
SELECT id,payment_provider,payment_intent_id,provider_refund_id,amount_minor,currency,
       status,skip_reason,attempts,run_at,last_error,created_at,updated_at
FROM commission_refund_intents_missing_failed;
DROP TABLE commission_refund_intents_missing_failed;

CREATE TABLE commission_entry_refunds (
    id TEXT PRIMARY KEY,
    entry_id TEXT NOT NULL REFERENCES referral_commission_entries(id),
    provider_refund_id TEXT NOT NULL,
    amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
    created_at TEXT NOT NULL,
    UNIQUE (entry_id,provider_refund_id)
);
CREATE INDEX idx_commission_entry_refunds_refund
    ON commission_entry_refunds(provider_refund_id);

ALTER TABLE referral_payouts RENAME TO referral_payouts_financial_migrate;
DROP INDEX IF EXISTS idx_referral_payouts_referrer;
DROP INDEX IF EXISTS idx_referral_payouts_status;
CREATE TABLE referral_payouts (
    id TEXT PRIMARY KEY,
    referrer_id TEXT NOT NULL,
    beneficiary_role TEXT NOT NULL CHECK (beneficiary_role IN ('kol','scout')),
    amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
    requested_amount_minor INTEGER NOT NULL CHECK (requested_amount_minor > 0),
    currency TEXT NOT NULL,
    method TEXT NOT NULL CHECK (method IN ('alipay','paypal')),
    account TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('requested','pending','approved','rejected','paid','failed')),
    payment_reference TEXT,
    rejection_reason TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    paid_at TEXT
);
CREATE INDEX idx_referral_payouts_beneficiary
    ON referral_payouts(referrer_id,beneficiary_role,status,created_at);
CREATE INDEX idx_referral_payouts_status
    ON referral_payouts(status,created_at);
INSERT INTO referral_payouts
    (id,referrer_id,beneficiary_role,amount_minor,requested_amount_minor,currency,
     method,account,status,payment_reference,rejection_reason,created_at,updated_at,paid_at)
SELECT p.id,p.referrer_id,
       COALESCE((
         SELECT e.beneficiary_role
         FROM referral_commission_entries e
         WHERE e.payout_id=p.id
         LIMIT 1
       ),'kol'),
       p.amount_minor,p.amount_minor,p.currency,p.method,p.account,p.status,
       p.payment_reference,p.rejection_reason,p.created_at,p.updated_at,p.paid_at
FROM referral_payouts_financial_migrate p;
DROP TABLE referral_payouts_financial_migrate;

CREATE TABLE referral_payout_adjustments (
    id TEXT PRIMARY KEY,
    payout_id TEXT NOT NULL REFERENCES referral_payouts(id),
    before_minor INTEGER NOT NULL CHECK (before_minor >= 0),
    after_minor INTEGER NOT NULL CHECK (after_minor >= 0),
    reason TEXT NOT NULL,
    actor TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX idx_payout_adjustments_payout
    ON referral_payout_adjustments(payout_id,created_at);
`); err != nil {
				return err
			}
			// 009 is also the encryption boundary for legacy payout accounts.
			rows, err := tx.QueryContext(ctx, `SELECT id,account FROM referral_payouts`)
			if err != nil {
				return err
			}
			type encryptedAccount struct{ id, account string }
			accounts := make([]encryptedAccount, 0)
			for rows.Next() {
				var item encryptedAccount
				if err := rows.Scan(&item.id, &item.account); err != nil {
					_ = rows.Close()
					return err
				}
				accounts = append(accounts, item)
			}
			if err := rows.Err(); err != nil {
				_ = rows.Close()
				return err
			}
			_ = rows.Close()
			for _, item := range accounts {
				sealed, sealErr := s.accounts.seal(item.account)
				if sealErr != nil {
					return sealErr
				}
				if _, err := tx.ExecContext(ctx, `UPDATE referral_payouts SET account=? WHERE id=?`, sealed, item.id); err != nil {
					return err
				}
			}
			return nil
		},
	}, {
		ID: "010_finance_tasks",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
CREATE TABLE finance_tasks (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL CHECK (type IN ('commission_refund_clawback')),
    provider TEXT NOT NULL,
    payment_intent_id TEXT NOT NULL,
    provider_refund_id TEXT NOT NULL,
    entry_id TEXT NOT NULL,
    beneficiary_id TEXT NOT NULL,
    beneficiary_role TEXT NOT NULL CHECK (beneficiary_role IN ('kol','scout')),
    amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
    currency TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending','resolved','rejected')),
    assignee TEXT,
    resolution_note TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    resolved_at TEXT,
    UNIQUE (type,provider_refund_id,entry_id)
);
CREATE INDEX idx_finance_tasks_pending
    ON finance_tasks(status,created_at);
CREATE INDEX idx_finance_tasks_refund
    ON finance_tasks(provider_refund_id);
`)
			return err
		},
	}, {
		ID: "011_finance_task_erasure",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
ALTER TABLE finance_tasks RENAME TO finance_tasks_erasure_migrate;
DROP INDEX IF EXISTS idx_finance_tasks_pending;
DROP INDEX IF EXISTS idx_finance_tasks_refund;
CREATE TABLE finance_tasks (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL CHECK (type IN ('commission_refund_clawback','user_erasure_review')),
    provider TEXT NOT NULL,
    payment_intent_id TEXT NOT NULL,
    provider_refund_id TEXT NOT NULL,
    entry_id TEXT NOT NULL,
    beneficiary_id TEXT NOT NULL,
    beneficiary_role TEXT NOT NULL CHECK (beneficiary_role IN ('kol','scout')),
    amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
    currency TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending','resolved','rejected')),
    assignee TEXT,
    resolution_note TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    resolved_at TEXT,
    UNIQUE (type,provider_refund_id,entry_id)
);
CREATE INDEX idx_finance_tasks_pending
    ON finance_tasks(status,created_at);
CREATE INDEX idx_finance_tasks_refund
    ON finance_tasks(provider_refund_id);
INSERT INTO finance_tasks
    (id,type,provider,payment_intent_id,provider_refund_id,entry_id,
     beneficiary_id,beneficiary_role,amount_minor,currency,status,assignee,
     resolution_note,created_at,updated_at,resolved_at)
SELECT id,type,provider,payment_intent_id,provider_refund_id,entry_id,
       beneficiary_id,beneficiary_role,amount_minor,currency,status,assignee,
       resolution_note,created_at,updated_at,resolved_at
FROM finance_tasks_erasure_migrate;
DROP TABLE finance_tasks_erasure_migrate;
`)
			return err
		},
	}}
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
		if _, err := s.db.ExecContext(ctx, `
INSERT INTO partner_roles (user_id,role,status,started_at,created_at,updated_at)
VALUES (?,'kol','active',?,?,?)
ON CONFLICT(user_id,role) DO UPDATE SET
  updated_at=excluded.updated_at`, userID, now, now, now); err != nil {
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
	return s.commissionTx(ctx, s.db, provider, intentID)
}

func (s *Store) commissionTx(ctx context.Context, runner interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, provider, intentID string) (Commission, error) {
	row := runner.QueryRowContext(ctx, `
SELECT id,payment_provider,payment_intent_id,business_order_id,product,buyer_user_id,
       referrer_id,recruiter_id,amount_minor,currency,referrer_rate_bps,
       referrer_commission_minor,recruiter_rate_bps,recruiter_commission_minor,
       status,refunded_minor,created_at,updated_at,paid_at
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
		&item.RecruiterAmount, &status, &item.RefundedMinor, &createdAt, &updatedAt, &paid,
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

func (s *Store) ListCommissions(ctx context.Context, referrerID string, limit int) ([]Commission, []BalanceSummary, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id,payment_provider,payment_intent_id,business_order_id,product,buyer_user_id,
       referrer_id,recruiter_id,amount_minor,currency,referrer_rate_bps,
       referrer_commission_minor,recruiter_rate_bps,recruiter_commission_minor,
       status,refunded_minor,created_at,updated_at,paid_at
FROM referral_commissions
WHERE referrer_id=? OR recruiter_id=?
ORDER BY created_at DESC LIMIT ?`, referrerID, referrerID, limit)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]Commission, 0)
	for rows.Next() {
		item, scanErr := scanCommission(rows.Scan)
		if scanErr != nil {
			return nil, nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	balanceRows, err := s.db.QueryContext(ctx, `
SELECT currency,
       COALESCE(SUM(amount_minor-refunded_minor),0),
       COALESCE(SUM(CASE WHEN status='pending' THEN amount_minor-refunded_minor ELSE 0 END),0),
       COALESCE(SUM(CASE WHEN status='paid' THEN amount_minor-refunded_minor ELSE 0 END),0)
FROM referral_commission_entries
WHERE beneficiary_id=? AND beneficiary_role='kol'
GROUP BY currency ORDER BY currency`, referrerID)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = balanceRows.Close() }()
	balances, err := scanBalanceSummaries(balanceRows)
	return items, balances, err
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
	commissions, balances, err := s.ListCommissions(ctx, referrerID, 100)
	if err != nil {
		return PartnerStats{}, err
	}
	attributed, err := s.CountAttributed(ctx, referrerID)
	if err != nil {
		return PartnerStats{}, err
	}
	return PartnerStats{
		Referrer: item, AttributedUsers: int(attributed), Commissions: commissions,
		Balances: balances,
	}, nil
}
