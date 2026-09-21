package referral

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrApplicationPending    = errors.New("referral: application already pending")
	ErrApplicationNotPending = errors.New("referral: application is not pending")
	ErrApplicationNotFound   = errors.New("referral: application not found")
	ErrPartnerNotFound       = errors.New("referral: partner not found")
	ErrInvalidPartnerStatus  = errors.New("referral: invalid partner status")
)

func (s *Store) SubmitScoutApplication(ctx context.Context, userID, note string) (PartnerApplication, error) {
	userID = strings.TrimSpace(userID)
	note = strings.TrimSpace(note)
	if note == "" {
		note = "no note"
	}
	referrer, err := s.ReferrerByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PartnerApplication{}, ErrNotEligible
		}
		return PartnerApplication{}, err
	}
	if referrer.Status != PartnerActive {
		return PartnerApplication{}, ErrNotEligible
	}
	if _, err := s.PendingApplicationByUser(ctx, userID); err == nil {
		return PartnerApplication{}, ErrApplicationPending
	} else if !errors.Is(err, sql.ErrNoRows) {
		return PartnerApplication{}, err
	}

	now := formatTime(s.nowFunc())
	id := s.idFunc()
	_, err = s.db.ExecContext(ctx, `
INSERT INTO referral_applications
(id,user_id,type,status,note,created_at,updated_at)
VALUES (?,?, 'scout',?,?,?,?)
ON CONFLICT(user_id) WHERE status='pending' DO NOTHING`,
		id, userID, string(ApplicationPending), note, now, now)
	if err != nil {
		return PartnerApplication{}, err
	}
	if item, err := s.PendingApplicationByUser(ctx, userID); err == nil {
		return item, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return PartnerApplication{}, err
	}
	return s.ApplicationByID(ctx, id)
}

func (s *Store) PendingApplicationByUser(ctx context.Context, userID string) (PartnerApplication, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT a.id,a.user_id,COALESCE(r.code,''),a.status,a.note,
       COALESCE(a.reviewer_id,''),COALESCE(a.decision_reason,''),
       a.created_at,a.reviewed_at
FROM referral_applications a
LEFT JOIN referrers r ON r.user_id=a.user_id
WHERE a.user_id=? AND a.status=?`, userID, string(ApplicationPending))
	return scanApplication(row.Scan)
}

func (s *Store) ApplicationByID(ctx context.Context, id string) (PartnerApplication, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT a.id,a.user_id,COALESCE(r.code,''),a.status,a.note,
       COALESCE(a.reviewer_id,''),COALESCE(a.decision_reason,''),
       a.created_at,a.reviewed_at
FROM referral_applications a
LEFT JOIN referrers r ON r.user_id=a.user_id
WHERE a.id=?`, id)
	return scanApplication(row.Scan)
}

func scanApplication(scan func(dest ...any) error) (PartnerApplication, error) {
	var item PartnerApplication
	var status, createdAt string
	var reviewer, decision sql.NullString
	var reviewed sql.NullString
	if err := scan(&item.ID, &item.UserID, &item.Code, &status, &item.Note,
		&reviewer, &decision, &createdAt, &reviewed); err != nil {
		return PartnerApplication{}, err
	}
	item.Status = ApplicationStatus(status)
	item.ReviewerID = reviewer.String
	item.DecisionReason = decision.String
	item.CreatedAt = parseTime(createdAt)
	if reviewed.Valid {
		t := parseTime(reviewed.String)
		item.ReviewedAt = &t
	}
	return item, nil
}

func (s *Store) ListApplications(ctx context.Context, status ApplicationStatus, limit int) ([]PartnerApplication, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	query := `
SELECT a.id,a.user_id,COALESCE(r.code,''),a.status,a.note,
       COALESCE(a.reviewer_id,''),COALESCE(a.decision_reason,''),
       a.created_at,a.reviewed_at
FROM referral_applications a
LEFT JOIN referrers r ON r.user_id=a.user_id`
	args := []any{}
	if status != "" {
		query += ` WHERE a.status=?`
		args = append(args, string(status))
	}
	query += ` ORDER BY a.created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]PartnerApplication, 0)
	for rows.Next() {
		item, err := scanApplication(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ListPartners is the admin-facing partner index. Empty filters mean all.
func (s *Store) ListPartners(ctx context.Context, role PartnerRole, status PartnerStatus, query string, limit int) ([]Referrer, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	query = strings.TrimSpace(query)
	clauses := make([]string, 0, 3)
	args := []any{}
	if role != "" {
		clauses = append(clauses, `role=?`)
		args = append(args, string(role))
	}
	if status != "" {
		clauses = append(clauses, `status=?`)
		args = append(args, string(status))
	}
	if query != "" {
		clauses = append(clauses, `(user_id LIKE ? OR code LIKE ?)`)
		like := "%" + query + "%"
		args = append(args, like, like)
	}
	sqlQuery := `SELECT user_id,code,role,status,recruiter_id,created_at,updated_at FROM referrers`
	if len(clauses) > 0 {
		sqlQuery += ` WHERE ` + strings.Join(clauses, ` AND `)
	}
	sqlQuery += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]Referrer, 0)
	for rows.Next() {
		item, err := scanReferrer(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) SetPartnerStatus(ctx context.Context, userID string, status PartnerStatus, actor, reason string) error {
	if status != PartnerActive && status != PartnerFrozen {
		return ErrInvalidPartnerStatus
	}
	if strings.TrimSpace(actor) == "" {
		return errors.New("referral: actor is required")
	}
	now := formatTime(s.nowFunc())
	res, err := s.db.ExecContext(ctx, `
UPDATE referrers SET status=?,updated_at=? WHERE user_id=?`,
		string(status), now, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrPartnerNotFound
	}
	if _, err := s.db.ExecContext(ctx, `
UPDATE partner_roles SET status=?,updated_at=?
WHERE user_id=? AND status IN ('active','frozen')`,
		string(status), now, userID); err != nil {
		return err
	}
	return s.recordAdminAction(ctx, actor, "partner.status_changed", "partner", userID,
		"", string(status), reason)
}

func (s *Store) DecideApplication(ctx context.Context, applicationID, actor string, approved bool, reason string) error {
	if strings.TrimSpace(actor) == "" {
		return errors.New("referral: actor is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var userID, status string
	err = tx.QueryRowContext(ctx, `
SELECT user_id,status FROM referral_applications WHERE id=?`, applicationID).Scan(&userID, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrApplicationNotFound
	}
	if err != nil {
		return err
	}
	if status != string(ApplicationPending) {
		return ErrApplicationNotPending
	}

	now := formatTime(s.nowFunc())
	decision := ApplicationRejected
	if approved {
		decision = ApplicationApproved
		res, err := tx.ExecContext(ctx, `
UPDATE referrers SET role=?,updated_at=? WHERE user_id=? AND status='active'`,
			string(RoleScout), now, userID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrNotEligible
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO partner_roles (user_id,role,status,started_at,created_at,updated_at)
VALUES (?,?,'active',?,?,?)
ON CONFLICT(user_id,role) DO UPDATE SET
  status='active',ended_at=NULL,updated_at=excluded.updated_at`,
			userID, string(RoleScout), now, now, now); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE referral_applications
SET status=?,reviewer_id=?,decision_reason=?,reviewed_at=?,updated_at=?
WHERE id=? AND status=?`,
		string(decision), actor, reason, now, now, applicationID, string(ApplicationPending)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO referral_admin_actions
(id,actor,action,target_type,target_id,before,after,reason,created_at)
VALUES (?,?,?,?,?,?,?,?,?)`,
		s.idFunc(), actor, "partner_application.decided", "partner_application", applicationID,
		ApplicationPending, decision, reason, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) recordAdminAction(ctx context.Context, actor, action, targetType, targetID, before, after, reason string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO referral_admin_actions
(id,actor,action,target_type,target_id,before,after,reason,created_at)
VALUES (?,?,?,?,?,?,?,?,?)`,
		s.idFunc(), actor, action, targetType, targetID, before, after, reason, formatTime(s.nowFunc()))
	if err != nil {
		return fmt.Errorf("referral: record admin action: %w", err)
	}
	return nil
}

func (s *Store) ListPartnerAdminActions(ctx context.Context, limit int) ([]PartnerAdminAction, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id,actor,action,target_type,target_id,COALESCE(before,''),
       COALESCE(after,''),COALESCE(reason,''),created_at
FROM referral_admin_actions ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]PartnerAdminAction, 0)
	for rows.Next() {
		var item PartnerAdminAction
		var createdAt string
		if err := rows.Scan(&item.ID, &item.Actor, &item.Action, &item.TargetType,
			&item.TargetID, &item.Before, &item.After, &item.Reason, &createdAt); err != nil {
			return nil, err
		}
		item.CreatedAt = parseTime(createdAt)
		items = append(items, item)
	}
	return items, rows.Err()
}

var _ = time.Time{}

func (s *Store) PromoteToScout(ctx context.Context, userID, actor, reason string) error {
	if strings.TrimSpace(actor) == "" || strings.TrimSpace(reason) == "" {
		return errors.New("referral: actor and reason are required")
	}
	now := formatTime(s.nowFunc())
	res, err := s.db.ExecContext(ctx, `
UPDATE referrers SET role=?,updated_at=?
WHERE user_id=? AND role='kol' AND status='active'`,
		string(RoleScout), now, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrPartnerNotFound
	}
	if _, err := s.db.ExecContext(ctx, `
INSERT INTO partner_roles (user_id,role,status,started_at,created_at,updated_at)
VALUES (?,?,'active',?,?,?)
ON CONFLICT(user_id,role) DO UPDATE SET
  status='active',ended_at=NULL,updated_at=excluded.updated_at`,
		userID, string(RoleScout), now, now, now); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `
UPDATE partner_roles SET status='retired',ended_at=?,updated_at=?
WHERE user_id=? AND role=? AND status='active'`,
		now, now, userID, string(RoleKOL)); err != nil {
		return err
	}
	return s.recordAdminAction(ctx, actor, "partner.promoted", "partner", userID,
		string(RoleKOL), string(RoleScout), reason)
}
