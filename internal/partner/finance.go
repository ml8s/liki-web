package partner

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/ml8s/liki-web/internal/metrics"
)

var (
	ErrFinanceTaskNotFound          = errors.New("partner: finance task not found")
	ErrInvalidFinanceTaskRequest    = errors.New("partner: invalid finance task request")
	ErrFinanceTaskInvalidTransition = errors.New("partner: finance task transition is not allowed")
)

func (s *Store) ListFinanceTasks(ctx context.Context, status FinanceTaskStatus, limit int) ([]FinanceTask, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	query := `
SELECT id,type,provider,payment_intent_id,provider_refund_id,entry_id,
       beneficiary_id,beneficiary_role,amount_minor,currency,status,
       COALESCE(assignee,''),COALESCE(resolution_note,''),created_at,updated_at,resolved_at
FROM finance_tasks`
	args := []any{}
	if status != "" {
		query += ` WHERE status=?`
		args = append(args, string(status))
	}
	query += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]FinanceTask, 0)
	for rows.Next() {
		item, err := scanFinanceTask(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ResolveFinanceTask(ctx context.Context, id, actor, status, note string) error {
	actor = strings.TrimSpace(actor)
	note = strings.TrimSpace(note)
	normalized := FinanceTaskStatus(strings.ToLower(strings.TrimSpace(status)))
	if actor == "" || note == "" ||
		(normalized != FinanceTaskResolved && normalized != FinanceTaskRejected) {
		return ErrInvalidFinanceTaskRequest
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var current string
	err = tx.QueryRowContext(ctx, `SELECT status FROM finance_tasks WHERE id=?`, id).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrFinanceTaskNotFound
	}
	if err != nil {
		return err
	}
	if current != string(FinanceTaskPending) {
		return ErrFinanceTaskInvalidTransition
	}
	now := formatTime(s.nowFunc())
	res, err := tx.ExecContext(ctx, `
UPDATE finance_tasks SET status=?,assignee=?,resolution_note=?,updated_at=?,resolved_at=?
WHERE id=? AND status=?`,
		string(normalized), actor, note, now, now, id, string(FinanceTaskPending))
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrFinanceTaskInvalidTransition
	}
	if err := recordReferralActionTx(ctx, tx, s.idFunc(), actor, "finance_task.resolved", "finance_task", id,
		string(FinanceTaskPending), string(normalized), note); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	metrics.Default.Inc("referral_finance_task_resolved_total", map[string]string{"result": string(normalized)})
	return nil
}

func scanFinanceTask(scan func(dest ...any) error) (FinanceTask, error) {
	var item FinanceTask
	var taskType, provider, paymentID, refundID, entryID, beneficiaryID, role, currency, status string
	var createdAt, updatedAt string
	var assignee, note, resolvedAt sql.NullString
	if err := scan(&item.ID, &taskType, &provider, &paymentID, &refundID, &entryID,
		&beneficiaryID, &role, &item.AmountMinor, &currency, &status,
		&assignee, &note, &createdAt, &updatedAt, &resolvedAt); err != nil {
		return FinanceTask{}, err
	}
	item.Type = FinanceTaskType(taskType)
	item.Provider = provider
	item.PaymentID = paymentID
	item.ProviderRefundID = refundID
	item.EntryID = entryID
	item.BeneficiaryID = beneficiaryID
	item.BeneficiaryRole = PartnerRole(role)
	item.Currency = currency
	item.Status = FinanceTaskStatus(status)
	item.Assignee = assignee.String
	item.ResolutionNote = note.String
	item.CreatedAt = parseTime(createdAt)
	item.UpdatedAt = parseTime(updatedAt)
	if resolvedAt.Valid {
		t := parseTime(resolvedAt.String)
		item.ResolvedAt = &t
	}
	return item, nil
}
