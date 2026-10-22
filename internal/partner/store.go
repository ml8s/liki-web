package partner

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ml8s/liki-web/internal/migrate"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

var (
	ErrInvalidCode  = errors.New("partner: invalid referral code")
	ErrSelfReferral = errors.New("partner: self referral is not allowed")
)

// referralCommissionProduct is the product sold by the referral programme.
// Payments routed through dodo or xunhu are eligible only after the product
// fulfillment succeeds.
const referralCommissionProduct = "liki-chat"

type Store struct {
	db               *sql.DB
	nowFunc          func() time.Time
	idFunc           func() string
	codeFunc         func() (string, error)
	accounts         *fieldSealer
	referrerRateBPS  int
	recruiterRateBPS int
}

func Open(path string, payoutSecret []byte) (*Store, error) {
	accounts, err := newFieldSealer(payoutSecret)
	if err != nil {
		return nil, err
	}
	db, err := sqliteutil.Open(path)
	if err != nil {
		return nil, err
	}
	return &Store{
		db:               db,
		accounts:         accounts,
		nowFunc:          func() time.Time { return time.Now().UTC() },
		idFunc:           uuid.NewString,
		codeFunc:         newPartnerCode,
		referrerRateBPS:  ReferrerRateBPS,
		recruiterRateBPS: RecruiterRateBPS,
	}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Migrate(ctx context.Context) error {
	return migrate.Run(ctx, s.db, s.migrations())
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

func scanPartner(scan func(dest ...any) error) (Partner, error) {
	var item Partner
	var role, status, createdAt, updatedAt string
	var recruiter sql.NullString
	if err := scan(&item.UserID, &item.Code, &role, &status, &recruiter, &createdAt, &updatedAt); err != nil {
		return Partner{}, err
	}
	item.Role = PartnerRole(role)
	item.Status = PartnerStatus(status)
	item.RecruiterID = recruiter.String
	item.CreatedAt = parseTime(createdAt)
	item.UpdatedAt = parseTime(updatedAt)
	return item, nil
}

func (s *Store) PartnerByUser(ctx context.Context, userID string) (Partner, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT user_id,code,role,status,recruiter_id,created_at,updated_at
FROM partners WHERE user_id=?`, userID)
	return scanPartner(row.Scan)
}

func (s *Store) PartnerByCode(ctx context.Context, code string) (Partner, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT user_id,code,role,status,recruiter_id,created_at,updated_at
FROM partners WHERE code=? AND status='active'`, strings.ToUpper(strings.TrimSpace(code)))
	return scanPartner(row.Scan)
}

func (s *Store) Attribute(ctx context.Context, userID, code string) (Attribution, error) {
	userID = strings.TrimSpace(userID)
	code = strings.ToUpper(strings.TrimSpace(code))
	if userID == "" || code == "" {
		return Attribution{}, ErrInvalidCode
	}
	referrer, err := s.PartnerByCode(ctx, code)
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
INSERT INTO partner_attributions (user_id,referrer_id,code,referred_at)
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
FROM partner_attributions WHERE user_id=?`, userID).Scan(
		&item.UserID, &item.ReferrerID, &item.Code, &referred)
	item.ReferredAt = parseTime(referred)
	return item, err
}

func (s *Store) AttributionForUser(ctx context.Context, userID string) (Attribution, bool, error) {
	var item Attribution
	var referred string
	err := s.db.QueryRowContext(ctx, `
SELECT user_id,referrer_id,code,referred_at
FROM partner_attributions WHERE user_id=?`, userID).Scan(
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
func (s *Store) commissionTx(ctx context.Context, runner interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, provider, paymentID string) (Commission, error) {
	row := runner.QueryRowContext(ctx, `
SELECT id,payment_provider,payment_intent_id,business_order_id,product,buyer_user_id,
       referrer_id,recruiter_id,amount_minor,currency,referrer_rate_bps,
       referrer_commission_minor,recruiter_rate_bps,recruiter_commission_minor,
       status,refunded_minor,created_at,updated_at,paid_at
FROM partner_commissions
WHERE payment_provider=? AND payment_intent_id=?`, provider, paymentID)
	return scanCommission(row.Scan)
}

func scanCommission(scan func(dest ...any) error) (Commission, error) {
	var item Commission
	var provider, status, currency, createdAt, updatedAt string
	var recruiter sql.NullString
	var paid sql.NullString
	if err := scan(
		&item.ID, &provider, &item.PaymentID, &item.BusinessOrderID, &item.Product,
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
FROM partner_commissions
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
FROM partner_commission_entries
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
SELECT count(*) FROM partner_attributions WHERE referrer_id=?`, referrerID).Scan(&n)
	return n, err
}

func (s *Store) PartnerStats(ctx context.Context, referrerID string) (PartnerStats, error) {
	item, err := s.PartnerByUser(ctx, referrerID)
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
		Partner: item, AttributedUsers: int(attributed), Commissions: commissions,
		Balances: balances,
	}, nil
}
