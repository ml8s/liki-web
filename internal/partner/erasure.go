package partner

import (
	"context"
	"errors"
	"strings"

	"github.com/ml8s/liki-web/internal/metrics"
)

var ErrPartnerFinancialHold = errors.New("partner: active payout requires finance settlement")

type ErasureResult struct {
	AlreadyErased bool
	PayoutsErased int64
	FinanceTasks  int64
}

// EraseUser freezes the partnership and erases contact/routing PII while
// retaining the pseudonymous financial ledger. Active payouts intentionally
// create a hold: their payment account is still necessary to fulfill money.
func (s *Store) EraseUser(ctx context.Context, userID string) (ErasureResult, error) {
	if strings.TrimSpace(userID) == "" {
		return ErasureResult{}, ErrPartnerNotFound
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ErasureResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var partnerCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM partners WHERE user_id=?`, userID).Scan(&partnerCount); err != nil {
		return ErasureResult{}, err
	}
	if partnerCount == 0 {
		return ErasureResult{AlreadyErased: true}, tx.Commit()
	}

	var activePayouts int
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(1) FROM partner_payouts
WHERE beneficiary_id=? AND status IN (?,?)`,
		userID, string(PayoutRequested), string(PayoutApproved)).Scan(&activePayouts); err != nil {
		return ErasureResult{}, err
	}
	if activePayouts > 0 {
		return ErasureResult{}, ErrPartnerFinancialHold
	}

	now := formatTime(s.nowFunc())
	erasureRef := "account-erasure"
	erasedPayouts, err := eraseCount(tx.ExecContext(ctx, `
UPDATE partner_payouts SET account='erased',updated_at=?
WHERE beneficiary_id=? AND account!='erased'`, now, userID))
	if err != nil {
		return ErasureResult{}, err
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE partners SET status='frozen',code=?,updated_at=?
WHERE user_id=?`, "erased-"+s.idFunc(), now, userID); err != nil {
		return ErasureResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE partner_roles SET status='frozen',ended_at=?,updated_at=?
WHERE user_id=? AND status='active'`, now, now, userID); err != nil {
		return ErasureResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE partner_applications
SET status=?,reviewer_id='system',decision_reason='account erasure',note='',reviewed_at=?,updated_at=?
WHERE user_id=? AND status=?`,
		string(ApplicationRejected), now, now, userID, string(ApplicationPending)); err != nil {
		return ErasureResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE partner_applications SET note='',updated_at=?
WHERE user_id=?`, now, userID); err != nil {
		return ErasureResult{}, err
	}

	rows, err := tx.QueryContext(ctx, `
SELECT id,beneficiary_role,amount_minor-refunded_minor,currency
FROM partner_commission_entries
WHERE beneficiary_id=? AND status=? AND amount_minor>refunded_minor`, userID, string(CommissionPending))
	if err != nil {
		return ErasureResult{}, err
	}
	type reviewTask struct {
		entryID, role, currency string
		amountMinor             int64
	}
	reviews := make([]reviewTask, 0)
	for rows.Next() {
		var item reviewTask
		if err := rows.Scan(&item.entryID, &item.role, &item.amountMinor, &item.currency); err != nil {
			_ = rows.Close()
			return ErasureResult{}, err
		}
		if item.amountMinor > 0 {
			reviews = append(reviews, item)
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return ErasureResult{}, err
	}
	_ = rows.Close()

	var createdTasks int64
	for _, item := range reviews {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO partner_finance_tasks
(id,type,provider,payment_intent_id,provider_refund_id,entry_id,
 beneficiary_id,beneficiary_role,amount_minor,currency,status,created_at,updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(type,provider_refund_id,entry_id) DO NOTHING`,
			s.idFunc(), string(FinanceTaskUserErasure), "system", "erasure:"+userID, erasureRef,
			item.entryID, userID, string(item.role), item.amountMinor, item.currency,
			string(FinanceTaskPending), now, now); err != nil {
			return ErasureResult{}, err
		}
		createdTasks++
	}

	if err := recordReferralActionTx(ctx, tx, s.idFunc(), "system", "partner.erased", "partner",
		userID, "active", "frozen", "account erasure"); err != nil {
		return ErasureResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ErasureResult{}, err
	}
	if createdTasks > 0 {
		metrics.Default.Inc("liki_web_referral_finance_task_created_total", map[string]string{"type": string(FinanceTaskUserErasure)})
	}
	return ErasureResult{PayoutsErased: erasedPayouts, FinanceTasks: createdTasks}, nil
}

func eraseCount(result interface{ RowsAffected() (int64, error) }, err error) (int64, error) {
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
