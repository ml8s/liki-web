package referral

import (
	"context"
	"database/sql"
)

func (s *Store) KOLOverview(ctx context.Context, userID string) (KOLOverview, error) {
	partner, err := s.KOLByID(ctx, userID)
	if err != nil {
		return KOLOverview{}, err
	}
	attributed, err := s.CountAttributed(ctx, partner.UserID)
	if err != nil {
		return KOLOverview{}, err
	}
	balances, err := s.kolBalances(ctx, partner.UserID)
	if err != nil {
		return KOLOverview{}, err
	}
	return KOLOverview{
		Referrer: partner, AttributedUsers: int(attributed), Balances: balances,
	}, nil
}

func (s *Store) kolBalances(ctx context.Context, userID string) ([]BalanceSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT currency,
       COALESCE(SUM(amount_minor-refunded_minor),0),
       COALESCE(SUM(CASE WHEN status='pending' THEN amount_minor-refunded_minor ELSE 0 END),0),
       COALESCE(SUM(CASE WHEN status='paid' THEN amount_minor-refunded_minor ELSE 0 END),0)
FROM referral_commission_entries
WHERE beneficiary_id=? AND beneficiary_role='kol'
GROUP BY currency
ORDER BY currency`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanBalanceSummaries(rows)
}

func (s *Store) ScoutRecruits(ctx context.Context, scoutID string) ([]ScoutRecruit, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT r.user_id,r.code,r.created_at,c.currency,
       COALESCE(SUM(e.amount_minor-e.refunded_minor),0),COUNT(c.id)
FROM referrers r
JOIN referral_commissions c ON c.referrer_id=r.user_id
JOIN referral_commission_entries e
  ON e.commission_id=c.id AND e.beneficiary_role='scout'
WHERE r.recruiter_id=? AND r.role='kol'
GROUP BY r.user_id,r.code,r.created_at,c.currency
ORDER BY r.created_at DESC,c.currency`, scoutID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	byUser := make(map[string]*ScoutRecruit)
	order := make([]string, 0)
	for rows.Next() {
		var userID, code, createdAt, currency string
		var amount, orderCount int64
		if err := rows.Scan(&userID, &code, &createdAt, &currency, &amount, &orderCount); err != nil {
			return nil, err
		}
		item, ok := byUser[userID]
		if !ok {
			item = &ScoutRecruit{UserID: userID, Code: code, CreatedAt: parseTime(createdAt)}
			byUser[userID] = item
			order = append(order, userID)
		}
		item.CommissionBalances = append(item.CommissionBalances, CurrencyAmount{Currency: currency, AmountMinor: amount})
		item.OrderCount += orderCount
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	items := make([]ScoutRecruit, 0, len(order))
	for _, userID := range order {
		items = append(items, *byUser[userID])
	}
	return items, nil
}

func (s *Store) ScoutOverview(ctx context.Context, userID string) (ScoutOverview, error) {
	partner, err := s.ScoutByID(ctx, userID)
	if err != nil {
		return ScoutOverview{}, err
	}
	recruits, err := s.ScoutRecruits(ctx, partner.UserID)
	if err != nil {
		return ScoutOverview{}, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT c.currency,
       COALESCE(SUM(c.amount_minor),0),
       COALESCE(SUM(e.amount_minor-e.refunded_minor),0),
       COALESCE(SUM(CASE WHEN e.status='pending' THEN e.amount_minor-e.refunded_minor ELSE 0 END),0),
       COALESCE(SUM(CASE WHEN e.status='paid' THEN e.amount_minor-e.refunded_minor ELSE 0 END),0)
FROM referral_commission_entries e
JOIN referral_commissions c ON c.id=e.commission_id
WHERE e.beneficiary_id=? AND e.beneficiary_role='scout'
GROUP BY c.currency
ORDER BY c.currency`, partner.UserID)
	if err != nil {
		return ScoutOverview{}, err
	}
	defer func() { _ = rows.Close() }()
	balances := make([]ScoutBalanceSummary, 0)
	for rows.Next() {
		var item ScoutBalanceSummary
		if err := rows.Scan(&item.Currency, &item.SalesMinor, &item.CommissionMinor,
			&item.PendingMinor, &item.PaidMinor); err != nil {
			return ScoutOverview{}, err
		}
		balances = append(balances, item)
	}
	if err := rows.Err(); err != nil {
		return ScoutOverview{}, err
	}
	var orderCount int64
	for _, recruit := range recruits {
		orderCount += recruit.OrderCount
	}
	return ScoutOverview{
		Partner: partner.Role, Recruits: recruits, RecruitCount: int64(len(recruits)),
		OrderCount: orderCount, Balances: balances,
	}, nil
}

func (s *Store) ListKOLCommissions(ctx context.Context, userID string, limit int) ([]Commission, []BalanceSummary, error) {
	partner, err := s.KOLByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	return s.listCommissionsForRole(ctx, partner.UserID, limit)
}

func (s *Store) ListScoutCommissions(ctx context.Context, userID string, limit int) ([]ScoutSale, []CurrencyAmount, error) {
	partner, err := s.ScoutByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT c.id,c.product,c.buyer_user_id,c.amount_minor,c.currency,
       e.amount_minor-e.refunded_minor,e.status,e.refunded_minor,c.created_at
FROM referral_commission_entries e
JOIN referral_commissions c ON c.id=e.commission_id
WHERE e.beneficiary_id=? AND e.beneficiary_role='scout'
ORDER BY c.created_at DESC LIMIT ?`, partner.UserID, limit)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]ScoutSale, 0)
	for rows.Next() {
		var item ScoutSale
		var status, createdAt string
		if err := rows.Scan(&item.OrderRef, &item.Product, &item.BuyerRef,
			&item.AmountMinor, &item.Currency, &item.ScoutCommissionMinor,
			&status, &item.RefundedMinor, &createdAt); err != nil {
			return nil, nil, err
		}
		item.Status = CommissionStatus(status)
		item.CreatedAt = parseTime(createdAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	balanceRows, err := s.db.QueryContext(ctx, `
SELECT currency,COALESCE(SUM(amount_minor-refunded_minor),0)
FROM referral_commission_entries
WHERE beneficiary_id=? AND beneficiary_role='scout'
GROUP BY currency ORDER BY currency`, partner.UserID)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = balanceRows.Close() }()
	balances, err := scanCurrencyAmounts(balanceRows)
	return items, balances, err
}

func (s *Store) listCommissionsForRole(ctx context.Context, userID string, limit int) ([]Commission, []BalanceSummary, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT c.id,c.payment_provider,c.payment_intent_id,c.business_order_id,c.product,c.buyer_user_id,
       c.referrer_id,c.recruiter_id,c.amount_minor,c.currency,c.referrer_rate_bps,
       e.amount_minor-e.refunded_minor,c.recruiter_rate_bps,c.recruiter_commission_minor,
       e.status,c.refunded_minor,c.created_at,c.updated_at,c.paid_at
FROM referral_commission_entries e
JOIN referral_commissions c ON c.id=e.commission_id
WHERE e.beneficiary_id=? AND e.beneficiary_role='kol'
ORDER BY c.created_at DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]Commission, 0)
	for rows.Next() {
		item, err := scanCommission(rows.Scan)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	balances, err := s.kolBalances(ctx, userID)
	return items, balances, err
}

func scanBalanceSummaries(rows *sql.Rows) ([]BalanceSummary, error) {
	items := make([]BalanceSummary, 0)
	for rows.Next() {
		var item BalanceSummary
		if err := rows.Scan(&item.Currency, &item.TotalMinor, &item.PendingMinor, &item.PaidMinor); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanCurrencyAmounts(rows *sql.Rows) ([]CurrencyAmount, error) {
	items := make([]CurrencyAmount, 0)
	for rows.Next() {
		var item CurrencyAmount
		if err := rows.Scan(&item.Currency, &item.AmountMinor); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
