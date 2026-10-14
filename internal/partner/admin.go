package partner

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrApplicationPending    = errors.New("partner: application already pending")
	ErrApplicationNotPending = errors.New("partner: application is not pending")
	ErrApplicationNotFound   = errors.New("partner: application not found")
	ErrPartnerNotFound       = errors.New("partner: partner not found")
	ErrInvalidPartnerStatus  = errors.New("partner: invalid partner status")
	ErrInvalidApplication    = errors.New("partner: invalid partner application")
	ErrAlreadyKOL            = errors.New("partner: user is already a KOL")
	ErrAlreadyScout          = errors.New("partner: user is already a scout")
	ErrRoleConflict          = errors.New("partner: KOL and Scout roles are separate")
)

// SubmitApplication records one of the two independent partner applications.
// Product consumption is never a paid entry ticket: eligibility is assessed by
// programme review, not by whether the applicant has purchased Liki Chat.
func (s *Store) SubmitApplication(ctx context.Context, role PartnerRole, userID, note string) (PartnerApplication, error) {
	userID = strings.TrimSpace(userID)
	note = strings.TrimSpace(note)
	role = normalizeRole(role)
	if userID == "" || (role != RoleKOL && role != RoleScout) {
		return PartnerApplication{}, ErrInvalidApplication
	}
	primary, err := s.PartnerByUser(ctx, userID)
	if err == nil {
		if primary.Role == role {
			if role == RoleKOL {
				return PartnerApplication{}, ErrAlreadyKOL
			}
			return PartnerApplication{}, ErrAlreadyScout
		}
		return PartnerApplication{}, ErrRoleConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return PartnerApplication{}, err
	}
	if _, err = s.PendingApplicationByUser(ctx, userID); err == nil {
		return PartnerApplication{}, ErrApplicationPending
	} else if !errors.Is(err, sql.ErrNoRows) {
		return PartnerApplication{}, err
	}

	now := formatTime(s.nowFunc())
	id := s.idFunc()
	if _, err := s.db.ExecContext(ctx, `
INSERT INTO partner_applications
(id,user_id,type,status,note,created_at,updated_at)
VALUES (?,?,?,?,?,?,?)
ON CONFLICT(user_id) WHERE status='pending' DO NOTHING`,
		id, userID, string(role), string(ApplicationPending), note, now, now); err != nil {
		return PartnerApplication{}, err
	}
	item, err := s.ApplicationByID(ctx, id)
	if err != nil {
		return PartnerApplication{}, err
	}
	return item, nil
}

func (s *Store) PendingApplicationByUser(ctx context.Context, userID string) (PartnerApplication, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT a.id,a.user_id,a.type,COALESCE(r.code,''),a.status,a.note,
       COALESCE(a.reviewer_id,''),COALESCE(a.decision_reason,''),
       a.created_at,a.reviewed_at
FROM partner_applications a
LEFT JOIN partners r ON r.user_id=a.user_id
WHERE a.user_id=? AND a.status=?`, userID, string(ApplicationPending))
	return scanApplication(row.Scan)
}

func (s *Store) LatestApplicationByUser(ctx context.Context, userID string, role PartnerRole) (PartnerApplication, bool, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT a.id,a.user_id,a.type,COALESCE(r.code,''),a.status,a.note,
       COALESCE(a.reviewer_id,''),COALESCE(a.decision_reason,''),
       a.created_at,a.reviewed_at
FROM partner_applications a
LEFT JOIN partners r ON r.user_id=a.user_id
WHERE a.user_id=? AND a.type=?
ORDER BY a.created_at DESC LIMIT 1`, userID, string(normalizeRole(role)))
	item, err := scanApplication(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return PartnerApplication{}, false, nil
	}
	if err != nil {
		return PartnerApplication{}, false, err
	}
	return item, true, nil
}

func (s *Store) ApplicationByID(ctx context.Context, id string) (PartnerApplication, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT a.id,a.user_id,a.type,COALESCE(r.code,''),a.status,a.note,
       COALESCE(a.reviewer_id,''),COALESCE(a.decision_reason,''),
       a.created_at,a.reviewed_at
FROM partner_applications a
LEFT JOIN partners r ON r.user_id=a.user_id
WHERE a.id=?`, id)
	return scanApplication(row.Scan)
}

func scanApplication(scan func(dest ...any) error) (PartnerApplication, error) {
	var item PartnerApplication
	var status, createdAt string
	var reviewer, decision sql.NullString
	var reviewed sql.NullString
	if err := scan(&item.ID, &item.UserID, &item.Type, &item.Code, &status, &item.Note,
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

func (s *Store) ListApplications(ctx context.Context, status ApplicationStatus, role PartnerRole, limit int) ([]PartnerApplication, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	query := `
SELECT a.id,a.user_id,a.type,COALESCE(r.code,''),a.status,a.note,
       COALESCE(a.reviewer_id,''),COALESCE(a.decision_reason,''),
       a.created_at,a.reviewed_at
FROM partner_applications a
LEFT JOIN partners r ON r.user_id=a.user_id`
	args := []any{}
	clauses := make([]string, 0, 2)
	if status != "" {
		clauses = append(clauses, `a.status=?`)
		args = append(args, string(status))
	}
	if role = normalizeRole(role); role != "" {
		clauses = append(clauses, `a.type=?`)
		args = append(args, string(role))
	}
	if len(clauses) > 0 {
		query += ` WHERE ` + strings.Join(clauses, ` AND `)
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
func (s *Store) ListPartners(ctx context.Context, role PartnerRole, status PartnerStatus, query string, limit int) ([]Partner, error) {
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
	sqlQuery := `SELECT user_id,code,role,status,recruiter_id,created_at,updated_at FROM partners`
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
	items := make([]Partner, 0)
	for rows.Next() {
		item, err := scanPartner(rows.Scan)
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
		return errors.New("partner: actor is required")
	}
	now := formatTime(s.nowFunc())
	res, err := s.db.ExecContext(ctx, `
UPDATE partners SET status=?,updated_at=? WHERE user_id=?`,
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
		return errors.New("partner: actor is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var userID, applicationType, status string
	err = tx.QueryRowContext(ctx, `
SELECT user_id,type,status FROM partner_applications WHERE id=?`, applicationID).
		Scan(&userID, &applicationType, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrApplicationNotFound
	}
	if err != nil {
		return err
	}
	if status != string(ApplicationPending) {
		return ErrApplicationNotPending
	}
	role := normalizeRole(PartnerRole(applicationType))
	if role != RoleKOL && role != RoleScout {
		return ErrInvalidApplication
	}

	now := formatTime(s.nowFunc())
	decision := ApplicationRejected
	if approved {
		decision = ApplicationApproved
		if err := ensurePartnerTx(ctx, tx, s, userID, role); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE partner_applications
SET status=?,reviewer_id=?,decision_reason=?,reviewed_at=?,updated_at=?
WHERE id=? AND status=?`,
		string(decision), actor, reason, now, now, applicationID, string(ApplicationPending)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO partner_admin_actions
(id,actor,action,target_type,target_id,before,after,reason,created_at)
VALUES (?,?,?,?,?,?,?,?,?)`,
		s.idFunc(), actor, "partner_application.decided", "partner_application", applicationID,
		ApplicationPending, decision, reason, now); err != nil {
		return err
	}
	return tx.Commit()
}

// ensurePartnerTx assigns one independent programme role. It never converts
// the other role. A KOL approved through a Scout's public link keeps that
// attribution so channel commission remains based on real recruitment.
func ensurePartnerTx(ctx context.Context, tx *sql.Tx, s *Store, userID string, role PartnerRole) error {
	var currentRole, currentStatus string
	err := tx.QueryRowContext(ctx,
		`SELECT role,status FROM partners WHERE user_id=?`, userID,
	).Scan(&currentRole, &currentStatus)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		recruiterID := ""
		if role == RoleKOL {
			var candidate string
			err := tx.QueryRowContext(ctx, `
SELECT r.user_id
FROM partner_attributions a
JOIN partners r ON r.user_id=a.referrer_id
JOIN partner_roles pr ON pr.user_id=r.user_id AND pr.role='scout' AND pr.status='active'
WHERE a.user_id=? AND a.referrer_id<>?`, userID, userID).Scan(&candidate)
			if err == nil {
				recruiterID = candidate
			} else if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}
		for range 5 {
			code, codeErr := s.codeFunc()
			if codeErr != nil {
				return codeErr
			}
			now := formatTime(s.nowFunc())
			res, insertErr := tx.ExecContext(ctx, `
INSERT INTO partners (user_id,code,role,status,recruiter_id,created_at,updated_at)
VALUES (?,?,?,?,?,?,?)
ON CONFLICT(user_id) DO NOTHING`,
				userID, code, string(role), string(PartnerActive), recruiterID, now, now)
			if insertErr != nil {
				return insertErr
			}
			if n, _ := res.RowsAffected(); n == 1 {
				break
			}
		}
		if err := tx.QueryRowContext(ctx,
			`SELECT role FROM partners WHERE user_id=?`, userID,
		).Scan(&currentRole); err != nil {
			return err
		}
		if currentRole != string(role) {
			return ErrRoleConflict
		}
	case err != nil:
		return err
	case currentRole != string(role) || currentStatus != string(PartnerActive):
		return ErrRoleConflict
	}

	now := formatTime(s.nowFunc())
	if _, err := tx.ExecContext(ctx, `
INSERT INTO partner_roles (user_id,role,status,started_at,created_at,updated_at)
VALUES (?,?,'active',?,?,?)
ON CONFLICT(user_id,role) DO UPDATE SET
  status='active',ended_at=NULL,updated_at=excluded.updated_at`,
		userID, string(role), now, now, now); err != nil {
		return err
	}
	return nil
}

func (s *Store) recordAdminAction(ctx context.Context, actor, action, targetType, targetID, before, after, reason string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO partner_admin_actions
(id,actor,action,target_type,target_id,before,after,reason,created_at)
VALUES (?,?,?,?,?,?,?,?,?)`,
		s.idFunc(), actor, action, targetType, targetID, before, after, reason, formatTime(s.nowFunc()))
	if err != nil {
		return fmt.Errorf("partner: record admin action: %w", err)
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
FROM partner_admin_actions ORDER BY created_at DESC LIMIT ?`, limit)
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

// CreateScoutPartner assigns the standalone Scout role. KOL and Scout are
// separate role assignments; this method never converts a KOL.
func (s *Store) CreateScoutPartner(ctx context.Context, userID, actor, reason string) (Partner, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(actor) == "" || strings.TrimSpace(reason) == "" {
		return Partner{}, errors.New("partner: user, actor, and reason are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Partner{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if err := ensurePartnerTx(ctx, tx, s, userID, RoleScout); err != nil {
		return Partner{}, err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO partner_admin_actions
(id,actor,action,target_type,target_id,before,after,reason,created_at)
VALUES (?,?,?,?,?,?,?,?,?)`,
		s.idFunc(), actor, "partner.scout_assigned", "partner", userID,
		"", string(RoleScout), reason, formatTime(s.nowFunc())); err != nil {
		return Partner{}, err
	}
	if err := tx.Commit(); err != nil {
		return Partner{}, err
	}
	return s.PartnerByUser(ctx, userID)
}
