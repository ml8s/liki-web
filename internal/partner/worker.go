package partner

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ml8s/liki-web/internal/metrics"
	"github.com/ml8s/liki-web/internal/payment"
)

type IntakeStatus string

const (
	IntakeQueued  IntakeStatus = "queued"
	IntakeRunning IntakeStatus = "running"
	IntakeSuccess IntakeStatus = "succeeded"
	IntakeFailed  IntakeStatus = "failed"
	IntakeSkipped IntakeStatus = "skipped"
)

const maxIntakeAttempts = 5

var ErrNoRunnableCommissionIntake = errors.New("partner: no runnable commission intake")

type CommissionIntake struct {
	ID              string
	Provider        string
	PaymentID       string
	BusinessOrderID string
	Product         string
	BuyerUserID     string
	AmountMinor     int64
	Currency        string
	Status          IntakeStatus
	Attempts        int
	RunAt           time.Time
	SkipReason      string
	LastError       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func classificationForPayment(sale payment.CommissionablePayment) (IntakeStatus, string) {
	if sale.Provider != payment.ProviderDodo && sale.Provider != payment.ProviderXunhu {
		return IntakeSkipped, "unsupported_provider"
	}
	if sale.Product != referralCommissionProduct {
		return IntakeSkipped, "ineligible_product"
	}
	if sale.UserID == "" || sale.AmountMinor <= 0 || sale.BusinessOrderID == "" {
		return IntakeSkipped, "invalid_payment_payload"
	}
	return IntakeQueued, ""
}

// EnqueueCommissionPayment durably records every fulfilled payment event. Rows
// that cannot earn a commission remain as auditable skips; eligible rows enter
// the local worker queue. This method never changes the payment domain state.
func (s *Store) EnqueueCommissionPayment(ctx context.Context, sale payment.CommissionablePayment) (IntakeStatus, string, error) {
	status, reason := classificationForPayment(sale)
	if status == IntakeQueued {
		attribution, attributed, err := s.AttributionForUser(ctx, sale.UserID)
		if err != nil {
			return IntakeQueued, "", fmt.Errorf("partner: load attribution for intake: %w", err)
		}
		if !attributed {
			status, reason = IntakeSkipped, "no_attribution"
		} else {
			referrer, err := s.PartnerByUser(ctx, attribution.ReferrerID)
			if err != nil {
				return IntakeQueued, "", fmt.Errorf("partner: load referrer for intake: %w", err)
			}
			if referrer.Status != PartnerActive {
				status, reason = IntakeSkipped, "referrer_not_active"
			} else if referrer.UserID == sale.UserID {
				status, reason = IntakeSkipped, "self_referral"
			}
		}
	}

	now := formatTime(s.nowFunc())
	id := s.idFunc()
	_, err := s.db.ExecContext(ctx, `
INSERT INTO partner_commission_intake
(id,payment_provider,payment_intent_id,business_order_id,product,buyer_user_id,
 amount_minor,currency,status,skip_reason,attempts,run_at,created_at,updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,0,?,?,?)
ON CONFLICT(payment_provider,payment_intent_id) DO NOTHING`,
		id, string(sale.Provider), sale.ID, sale.BusinessOrderID, sale.Product,
		sale.UserID, sale.AmountMinor, strings.ToUpper(sale.Currency), string(status),
		reason, now, now, now)
	if err != nil {
		return status, reason, fmt.Errorf("partner: enqueue commission intent: %w", err)
	}
	metrics.Default.Inc("liki_web_referral_commission_intake_total", map[string]string{"status": string(status)})
	return status, reason, nil
}

// ReconcileSuccessfulPayments is the compensating read from the payment bounded
// context. It covers process crashes between fulfillment success and referral
// intake, and requeues skips whose attribution arrived after payment.
func (s *Store) ReconcileSuccessfulPayments(ctx context.Context, platformDB *sql.DB, limit int) (int, error) {
	if platformDB == nil {
		return 0, errors.New("partner: platform database is required")
	}
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := platformDB.QueryContext(ctx, `
SELECT pi.id,pi.business_order_id,pi.user_id,pi.product,pi.provider,
       pi.amount_minor,pi.currency
FROM payment_intents AS pi
JOIN payment_completion_tasks AS ft ON ft.payment_intent_id=pi.id
WHERE pi.status=? AND ft.status=?
ORDER BY pi.updated_at ASC LIMIT ?`, string(payment.IntentPaid), "succeeded", limit)
	if err != nil {
		return 0, fmt.Errorf("partner: query successful payments: %w", err)
	}
	defer func() { _ = rows.Close() }()

	enqueued := 0
	for rows.Next() {
		sale := payment.CommissionablePayment{}
		var provider string
		if err := rows.Scan(&sale.ID, &sale.BusinessOrderID, &sale.UserID,
			&sale.Product, &provider, &sale.AmountMinor, &sale.Currency); err != nil {
			return enqueued, err
		}
		sale.Provider = payment.Provider(provider)
		if _, _, err := s.EnqueueCommissionPayment(ctx, sale); err != nil {
			return enqueued, err
		}
		enqueued++
	}
	if err := rows.Err(); err != nil {
		return enqueued, err
	}
	if _, err := s.db.ExecContext(ctx, `
UPDATE partner_commission_intake AS ci
SET status=?,skip_reason=NULL,run_at=?,updated_at=?
WHERE status=? AND skip_reason='no_attribution'
  AND EXISTS (SELECT 1 FROM partner_attributions AS a WHERE a.user_id=ci.buyer_user_id)`,
		string(IntakeQueued), formatTime(s.nowFunc()), formatTime(s.nowFunc()), string(IntakeSkipped)); err != nil {
		return enqueued, fmt.Errorf("partner: requeue attributed skips: %w", err)
	}
	return enqueued, nil
}

func (s *Store) claimCommissionIntake(ctx context.Context, now time.Time) (CommissionIntake, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CommissionIntake{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var item CommissionIntake
	var provider, paymentID, businessOrderID, product, buyer, currency string
	var amountMinor, attempts int64
	var status, runAt, createdAt, updatedAt string
	var skipReason, lastError sql.NullString
	err = tx.QueryRowContext(ctx, `
SELECT id,payment_provider,payment_intent_id,business_order_id,product,buyer_user_id,
       amount_minor,currency,status,attempts,run_at,skip_reason,last_error,created_at,updated_at
FROM partner_commission_intake
WHERE status=? AND run_at<=?
ORDER BY run_at LIMIT 1`, string(IntakeQueued), formatTime(now)).Scan(
		&item.ID, &provider, &paymentID, &businessOrderID, &product,
		&buyer, &amountMinor, &currency, &status, &attempts,
		&runAt, &skipReason, &lastError, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return CommissionIntake{}, ErrNoRunnableCommissionIntake
	}
	if err != nil {
		return CommissionIntake{}, err
	}
	res, err := tx.ExecContext(ctx, `
UPDATE partner_commission_intake SET status=?,attempts=attempts+1,updated_at=?
WHERE id=? AND status=?`, string(IntakeRunning), formatTime(now), item.ID, string(IntakeQueued))
	if err != nil {
		return CommissionIntake{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return CommissionIntake{}, ErrNoRunnableCommissionIntake
	}
	if err := tx.Commit(); err != nil {
		return CommissionIntake{}, err
	}

	item = CommissionIntake{
		ID: item.ID, Provider: provider, PaymentID: paymentID,
		BusinessOrderID: businessOrderID, Product: product, BuyerUserID: buyer,
		AmountMinor: amountMinor, Currency: currency, Status: IntakeRunning,
		Attempts: int(attempts) + 1, RunAt: parseTime(runAt), SkipReason: skipReason.String,
		LastError: lastError.String, CreatedAt: parseTime(createdAt), UpdatedAt: now,
	}
	return item, nil
}

func nullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func (s *Store) markIntakeSucceeded(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE partner_commission_intake SET status=?,last_error=NULL,updated_at=?
WHERE id=?`, string(IntakeSuccess), formatTime(s.nowFunc()), id)
	return err
}

func (s *Store) markIntakeFailed(ctx context.Context, id string, failure error) error {
	attempts, err := s.intakeAttempts(ctx, id)
	if err != nil {
		return err
	}
	status := IntakeQueued
	if attempts >= maxIntakeAttempts {
		status = IntakeFailed
	}
	runAt := s.nowFunc()
	if status == IntakeQueued {
		delays := []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour, 4 * time.Hour}
		runAt = runAt.Add(delays[min(attempts, len(delays)-1)])
	}
	_, err = s.db.ExecContext(ctx, `
UPDATE partner_commission_intake SET status=?,run_at=?,last_error=?,updated_at=? WHERE id=?`,
		string(status), formatTime(runAt), failure.Error(), formatTime(s.nowFunc()), id)
	return err
}

func (s *Store) intakeAttempts(ctx context.Context, id string) (int, error) {
	var attempts int
	err := s.db.QueryRowContext(ctx,
		`SELECT attempts FROM partner_commission_intake WHERE id=?`, id).Scan(&attempts)
	return attempts, err
}

func (s *Store) settleIntake(ctx context.Context, item CommissionIntake) error {
	attribution, ok, err := s.AttributionForUser(ctx, item.BuyerUserID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("partner: attribution disappeared for buyer %s", item.BuyerUserID)
	}
	referrer, err := s.PartnerByUser(ctx, attribution.ReferrerID)
	if err != nil {
		return err
	}
	if referrer.Status != PartnerActive || referrer.UserID == item.BuyerUserID {
		return s.markIntakeSkipped(ctx, item.ID, "referrer_not_eligible")
	}
	if referrer.Role != RoleKOL && referrer.Role != RoleScout {
		return s.markIntakeSkipped(ctx, item.ID, "referrer_role_not_eligible")
	}

	beneficiaryRole := RoleKOL
	if referrer.Role == RoleScout {
		beneficiaryRole = RoleScout
	}
	recruiterID, recruiterRate, recruiterAmount := "", 0, int64(0)
	if referrer.Role == RoleKOL && referrer.RecruiterID != "" &&
		referrer.RecruiterID != referrer.UserID && referrer.RecruiterID != item.BuyerUserID {
		scout, scoutErr := s.ScoutByID(ctx, referrer.RecruiterID)
		if scoutErr != nil && !errors.Is(scoutErr, sql.ErrNoRows) {
			return scoutErr
		}
		if scoutErr == nil && scout.Status == PartnerActive {
			recruiterID = scout.UserID
			recruiterRate = s.recruiterRateBPS
			recruiterAmount = commission_parts(item.AmountMinor, recruiterRate)
		}
	}
	now := formatTime(s.nowFunc())
	commission := Commission{
		ID: s.idFunc(), PaymentProvider: item.Provider, PaymentID: item.PaymentID,
		BusinessOrderID: item.BusinessOrderID, Product: item.Product, BuyerUserID: item.BuyerUserID,
		ReferrerID: referrer.UserID, RecruiterID: recruiterID,
		BeneficiaryRole: beneficiaryRole,
		AmountMinor:     item.AmountMinor, Currency: item.Currency,
		ReferrerRateBPS: s.referrerRateBPS, ReferrerAmount: commission_parts(item.AmountMinor, s.referrerRateBPS),
		RecruiterRateBPS: recruiterRate, RecruiterAmount: recruiterAmount,
		Status: CommissionPending, CreatedAt: parseTime(now), UpdatedAt: parseTime(now),
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `
INSERT INTO partner_commissions
(id,payment_provider,payment_intent_id,business_order_id,product,buyer_user_id,
 referrer_id,recruiter_id,amount_minor,currency,referrer_rate_bps,
 referrer_commission_minor,recruiter_rate_bps,recruiter_commission_minor,
 status,created_at,updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(payment_provider,payment_intent_id) DO NOTHING`,
		commission.ID, commission.PaymentProvider, commission.PaymentID,
		commission.BusinessOrderID, commission.Product, commission.BuyerUserID,
		commission.ReferrerID, nullable(commission.RecruiterID), commission.AmountMinor,
		commission.Currency, commission.ReferrerRateBPS, commission.ReferrerAmount,
		commission.RecruiterRateBPS, commission.RecruiterAmount,
		string(commission.Status), now, now)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if commission, err = s.commissionTx(ctx, tx, item.Provider, item.PaymentID); err != nil {
			return err
		}
	}
	if err := ensureCommissionEntriesTx(ctx, tx, commission, now); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return s.markIntakeSucceeded(ctx, item.ID)
}

func ensureCommissionEntriesTx(ctx context.Context, tx *sql.Tx, commission Commission, now string) error {
	entries := []CommissionEntry{{
		ID: commission.ID + ":" + string(commission.BeneficiaryRole), CommissionID: commission.ID,
		BeneficiaryID: commission.ReferrerID, BeneficiaryRole: commission.BeneficiaryRole,
		AmountMinor: commission.ReferrerAmount, Currency: commission.Currency,
		Status: commission.Status, CreatedAt: commission.CreatedAt, UpdatedAt: commission.UpdatedAt,
	}}
	if commission.RecruiterID != "" && commission.RecruiterAmount > 0 {
		entries = append(entries, CommissionEntry{
			ID: commission.ID + ":scout", CommissionID: commission.ID,
			BeneficiaryID: commission.RecruiterID, BeneficiaryRole: RoleScout,
			AmountMinor: commission.RecruiterAmount, Currency: commission.Currency,
			Status: commission.Status, CreatedAt: commission.CreatedAt, UpdatedAt: commission.UpdatedAt,
		})
	}
	for _, entry := range entries {
		if entry.AmountMinor <= 0 || entry.BeneficiaryID == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO partner_commission_entries
(id,commission_id,beneficiary_id,beneficiary_role,amount_minor,refunded_minor,
 currency,status,created_at,updated_at)
VALUES (?,?,?,?,?,0,?,?,?,?)
ON CONFLICT(commission_id,beneficiary_role) DO NOTHING`,
			entry.ID, entry.CommissionID, entry.BeneficiaryID, string(entry.BeneficiaryRole),
			entry.AmountMinor, entry.Currency, string(entry.Status), now, now); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) markIntakeSkipped(ctx context.Context, id, reason string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE partner_commission_intake SET status=?,skip_reason=?,updated_at=? WHERE id=?`,
		string(IntakeSkipped), reason, formatTime(s.nowFunc()), id)
	return err
}

// ProcessCommissionPayments settles at most limit durable referral events. The
// referral commission row and intake completion are not a cross-database
// transaction; conflict-safe inserts plus retry make the operation effectively
// exactly-once.
func (s *Store) ProcessCommissionPayments(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 1
	}
	processed := 0
	for processed < limit {
		item, err := s.claimCommissionIntake(ctx, s.nowFunc())
		if errors.Is(err, ErrNoRunnableCommissionIntake) {
			return processed, nil
		}
		if err != nil {
			return processed, err
		}
		taskCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		if err := s.settleIntake(taskCtx, item); err != nil {
			cancel()
			if markErr := s.markIntakeFailed(ctx, item.ID, err); markErr != nil {
				return processed, markErr
			}
			slog.WarnContext(ctx, "partner: commission intake retry", "id", item.ID,
				"buyer", item.BuyerUserID, "attempt", item.Attempts, "err", err)
			metrics.Default.Inc("liki_web_referral_commission_worker_total", map[string]string{"result": "retry_or_failed"})
			continue
		}
		cancel()
		metrics.Default.Inc("liki_web_referral_commission_settled_total", nil)
		metrics.Default.Inc("liki_web_referral_commission_worker_total", map[string]string{"result": "succeeded"})
		processed++
	}
	return processed, nil
}

func (s *Store) recoverCommissionIntakes(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE partner_commission_intake SET status=?,run_at=?,updated_at=? WHERE status=?`,
		string(IntakeQueued), formatTime(s.nowFunc()), formatTime(s.nowFunc()), string(IntakeRunning))
	return err
}

// recoverCommissionRefunds mirrors recoverCommissionIntakes for the refund
// side: a restart between claim and apply would otherwise leave the intake
// stuck in running forever.
func (s *Store) recoverCommissionRefunds(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE partner_commission_refund_intake SET status=?,run_at=?,updated_at=? WHERE status=?`,
		string(IntakeQueued), formatTime(s.nowFunc()), formatTime(s.nowFunc()), string(IntakeRunning))
	return err
}

// StartCommissionWorker owns the referral side of the payment -> referral
// boundary. Payment failures never enter here; only durable fulfilled facts do.
// SetCommissionRates 由部署配置注入分成比例（BPS）；未调用时用内置默认。
func (s *Store) SetCommissionRates(referrerBPS, recruiterBPS int) {
	s.referrerRateBPS = referrerBPS
	s.recruiterRateBPS = recruiterBPS
}

func (s *Store) StartCommissionWorker(ctx context.Context, platformDB *sql.DB, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	// 部署可通过 env 覆盖分成比例（BPS）；未配置时用内置默认。
	if s.referrerRateBPS == 0 {
		s.referrerRateBPS = ReferrerRateBPS
	}
	if s.recruiterRateBPS == 0 {
		s.recruiterRateBPS = RecruiterRateBPS
	}
	if err := s.recoverCommissionIntakes(ctx); err != nil {
		slog.ErrorContext(ctx, "partner: recover commission intakes", "err", err)
	}
	if err := s.recoverCommissionRefunds(ctx); err != nil {
		slog.ErrorContext(ctx, "partner: recover commission refunds", "err", err)
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if n, err := s.ProcessCommissionPayments(ctx, 100); err != nil {
			slog.ErrorContext(ctx, "partner: process commissions", "err", err)
		} else if n > 0 {
			slog.InfoContext(ctx, "partner: commissions settled", "count", n)
		}
		if n, err := s.ProcessCommissionRefunds(ctx, 100); err != nil {
			slog.ErrorContext(ctx, "partner: process refunds", "err", err)
		} else if n > 0 {
			slog.InfoContext(ctx, "partner: commission refunds applied", "count", n)
		}
		if n, err := s.ReconcileSuccessfulPayments(ctx, platformDB, 200); err != nil {
			slog.ErrorContext(ctx, "partner: reconcile payments", "err", err)
		} else if n > 0 {
			slog.DebugContext(ctx, "partner: reconciled successful payments", "count", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// VoidCommissionForPayment makes an unpaid commission unusable after a refund.
// Paid commissions are intentionally not mutated; finance must handle clawback.
func (s *Store) VoidCommissionForPayment(ctx context.Context, provider payment.Provider, paymentID, reason string) error {
	providerName := string(provider)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	now := formatTime(s.nowFunc())
	_, err = tx.ExecContext(ctx, `
UPDATE partner_commission_entries
SET status=?,updated_at=?
WHERE commission_id=(SELECT id FROM partner_commissions WHERE payment_provider=? AND payment_intent_id=?)
  AND status=? AND payout_id IS NULL`,
		string(CommissionVoided), now, providerName, paymentID, string(CommissionPending))
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE partner_commissions
SET status=?,updated_at=?
WHERE payment_provider=? AND payment_intent_id=? AND status=?`,
		string(CommissionVoided), now, providerName, paymentID, string(CommissionPending)); err != nil {
		return err
	}
	if err := recordReferralActionTx(ctx, tx, s.idFunc(), "system", "commission.voided", "commission",
		providerName+":"+paymentID, string(CommissionPending), string(CommissionVoided), reason); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	metrics.Default.Inc("liki_web_referral_commission_void_total", map[string]string{"provider": providerName})
	return nil
}
