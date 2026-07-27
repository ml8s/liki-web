package feedback //nolint:errcheck

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Feedback struct {
	ID        string `json:"id"`
	Category  string `json:"category"`          // workflow | api | doc | bug | feature | llm_self | other
	Message   string `json:"message"`
	Context   string `json:"context,omitempty"` // JSON — 当前上下文
	Status    string `json:"status"`            // pending | processing | done | rejected
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
			id        TEXT PRIMARY KEY,
			category  TEXT NOT NULL DEFAULT 'other',
			message   TEXT NOT NULL,
			context   TEXT NOT NULL DEFAULT '',
			status    TEXT NOT NULL DEFAULT 'pending',
			issue_url TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)
	`)
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
		INSERT INTO feedback (id, category, message, context, status, issue_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, fb.ID, fb.Category, fb.Message, fb.Context, fb.Status, fb.IssueURL, fb.CreatedAt, fb.UpdatedAt)
	return err
}

func (s *Store) List(status string, limit int) ([]Feedback, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.db.Query(`
		SELECT id, category, message, context, status, issue_url, created_at, updated_at
		FROM feedback
		WHERE (? = '' OR status = ?)
		ORDER BY created_at ASC
		LIMIT ?
	`, status, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Feedback
	for rows.Next() {
		var fb Feedback
		if err := rows.Scan(&fb.ID, &fb.Category, &fb.Message, &fb.Context, &fb.Status, &fb.IssueURL, &fb.CreatedAt, &fb.UpdatedAt); err != nil {
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
