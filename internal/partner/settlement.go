package partner

import "context"

// SettlementForRole returns the money-only view allowed for active and retired
// role capabilities. It never exposes dashboard or recruiting data, so a
// retired participant can settle history without reguring programme access.
func (s *Store) SettlementForRole(ctx context.Context, userID string, role PartnerRole) (Settlement, error) {
	role = normalizeRole(role)
	if role != RoleKOL && role != RoleScout {
		return Settlement{}, ErrInvalidPayoutRequest
	}
	partner, err := s.PartnerForPayout(ctx, userID, role)
	if err != nil {
		return Settlement{}, err
	}
	status, _, err := s.ProgrammeRoleStatus(ctx, userID, role)
	if err != nil {
		return Settlement{}, err
	}
	balances, err := s.roleBalances(ctx, partner.UserID, role)
	if err != nil {
		return Settlement{}, err
	}
	payouts, err := s.ListPayoutsForRole(ctx, partner.UserID, role, 100)
	if err != nil {
		return Settlement{}, err
	}
	return Settlement{
		Role: role, Status: status, Balances: balances, Payouts: payouts,
	}, nil
}

func (s *Store) roleBalances(ctx context.Context, userID string, role PartnerRole) ([]BalanceSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT currency,
       COALESCE(SUM(amount_minor-refunded_minor),0),
       COALESCE(SUM(CASE WHEN status='pending' THEN amount_minor-refunded_minor ELSE 0 END),0),
       COALESCE(SUM(CASE WHEN status='paid' THEN amount_minor-refunded_minor ELSE 0 END),0)
FROM referral_commission_entries
WHERE beneficiary_id=? AND beneficiary_role=?
GROUP BY currency
ORDER BY currency`, userID, string(role))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanBalanceSummaries(rows)
}
