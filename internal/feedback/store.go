package feedback

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Feedback stores the raw feedback-v1 JSON payload as-is plus backend-managed
// lifecycle fields. The payload is self-describing (schema_version inside), so
// field additions in the skill schema never require backend changes.
type Feedback struct {
	ID        string `json:"id"`
	Payload   string `json:"payload"` // 原文 JSON，后端不解字段
	Hash      string `json:"-"`       // sha256(payload)，字段无关去重键
	Status    string `json:"status"`  // pending | processing | done | rejected
	IssueURL  string `json:"issue_url,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("feedback migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS feedback (
			id          TEXT PRIMARY KEY,
			payload     TEXT NOT NULL DEFAULT '',
			hash        TEXT NOT NULL DEFAULT '',
			status      TEXT NOT NULL DEFAULT 'pending',
			issue_url   TEXT NOT NULL DEFAULT '',
			created_at  TEXT NOT NULL,
			updated_at  TEXT NOT NULL
		)
	`)
	if err != nil {
		return err
	}
	// legacy 库（旧 category/message/context 结构）：补齐 payload/hash 列。
	for _, col := range []struct{ name, def string }{
		{"payload", "''"}, {"hash", "''"},
	} {
		if err := s.addColumnIfMissing(col.name, col.def); err != nil {
			return err
		}
	}
	// legacy 行回填为 raw JSON，便于排查（旧列保留不用）。
	_, _ = s.db.Exec(`
		UPDATE feedback SET
			payload = json_object('schema_version','legacy','category',category,'message',message)
		WHERE payload = '' AND message != ''
	`)
	return nil
}

func (s *Store) addColumnIfMissing(name, def string) error {
	var found bool
	rows, err := s.db.Query(`SELECT name FROM pragma_table_info('feedback') WHERE name = ?`, name)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		found = true
	}
	if found {
		return nil
	}
	_, err = s.db.Exec(`ALTER TABLE feedback ADD COLUMN ` + name + ` TEXT NOT NULL DEFAULT ` + def)
	return err
}

func (s *Store) Create(fb *Feedback) error {
	fb.ID = uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)
	fb.CreatedAt = now
	fb.UpdatedAt = now
	if fb.Status == "" {
		fb.Status = "pending"
	}
	_, err := s.db.Exec(`
		INSERT INTO feedback (id, payload, hash, status, issue_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, fb.ID, fb.Payload, fb.Hash, fb.Status, fb.IssueURL, fb.CreatedAt, fb.UpdatedAt)
	return err
}

// ExistingByHash returns (id, status) when an identical payload already exists.
// Content-hash dedup is field-agnostic: it stays correct however the feedback
// schema evolves.
func (s *Store) ExistingByHash(hash string) (string, string, bool) {
	var id, status string
	err := s.db.QueryRow(`SELECT id, status FROM feedback WHERE hash = ? LIMIT 1`, hash).Scan(&id, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", false
	}
	if err != nil {
		return "", "", false
	}
	return id, status, true
}

func (s *Store) List(status string, limit int) ([]Feedback, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.db.Query(`
		SELECT id, payload, status, issue_url, created_at, updated_at
		FROM feedback
		WHERE (? = '' OR status = ?)
		ORDER BY created_at ASC
		LIMIT ?
	`, status, status, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := []Feedback{} // 空结果序列化为 [] 而非 null
	for rows.Next() {
		var fb Feedback
		if err := rows.Scan(&fb.ID, &fb.Payload, &fb.Status, &fb.IssueURL, &fb.CreatedAt, &fb.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, fb)
	}
	return out, rows.Err()
}

func (s *Store) Update(id, status, issueURL string) error {
	res, err := s.db.Exec(`
		UPDATE feedback SET status = ?, issue_url = ?, updated_at = ?
		WHERE id = ?
	`, status, issueURL, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("feedback %s: not found", id)
	}
	return nil
}
