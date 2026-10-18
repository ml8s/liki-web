// Package records persists user tool results (liuyao casts, huangli ranges,
// bond and bazhai previews) so the service pages can offer a history list.
// Records are opaque JSON snapshots: the engine contract stays the single
// source of truth for their shape.
package records

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/ml8s/liki-web/internal/migrate"
)

var ErrNotFound = errors.New("records: record not found")

// ErrInvalid marks caller-fixable input problems (kind/title/JSON shape) so the
// HTTP layer can answer 400 without masking store failures as validation.
var ErrInvalid = errors.New("records: invalid input")

// MaxTitleRunes bounds record titles, matching the session rename contract.
const MaxTitleRunes = 60

type Kind string

const (
	KindLiuyao  Kind = "liuyao"
	KindHuangli Kind = "huangli"
	KindBond    Kind = "bond"
	KindBazhai  Kind = "bazhai"
)

var kinds = map[Kind]struct{}{
	KindLiuyao:  {},
	KindHuangli: {},
	KindBond:    {},
	KindBazhai:  {},
}

func (k Kind) Valid() bool {
	_, ok := kinds[k]
	return ok
}

type Record struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	Kind      Kind            `json:"kind"`
	Title     string          `json:"title"`
	Input     json.RawMessage `json:"input"`
	Output    json.RawMessage `json:"output,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type CreateInput struct {
	Kind   Kind
	Title  string
	Input  json.RawMessage
	Output json.RawMessage
}

func (in CreateInput) Validate() error {
	if !in.Kind.Valid() {
		return errors.New("records: unknown kind")
	}
	title := strings.TrimSpace(in.Title)
	if title == "" || utf8.RuneCountInString(title) > MaxTitleRunes {
		return errors.New("records: title must be 1-60 characters")
	}
	if len(in.Input) == 0 || in.Input[0] != '{' || !json.Valid(in.Input) {
		return errors.New("records: input must be a JSON object")
	}
	if len(in.Output) > 0 && (!json.Valid(in.Output) || in.Output[0] != '{') {
		return errors.New("records: output must be a JSON object")
	}
	return nil
}

type Store struct {
	db      *sql.DB
	nowFunc func() time.Time
	idFunc  func() string
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db, nowFunc: func() time.Time { return time.Now().UTC() }, idFunc: uuid.NewString}
}

func (s *Store) Migrate(ctx context.Context) error {
	return migrate.Run(ctx, s.db, []migrate.Migration{
		{
			ID: "001_tool_records",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS tool_records (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('liuyao','huangli','bond','bazhai')),
    title TEXT NOT NULL,
    input TEXT NOT NULL,
    output TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_tool_records_user_kind
    ON tool_records(user_id, kind, deleted_at, updated_at DESC);
`)
				return err
			},
		},
	})
}

func scanRecord(scan func(dest ...any) error) (Record, error) {
	var record Record
	var kind, input, createdAt, updatedAt string
	var output sql.NullString
	if err := scan(&record.ID, &record.UserID, &kind, &record.Title, &input, &output, &createdAt, &updatedAt); err != nil {
		return Record{}, err
	}
	record.Kind = Kind(kind)
	record.Input = json.RawMessage(input)
	if output.Valid {
		record.Output = json.RawMessage(output.String)
	}
	var err error
	if record.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
		return Record{}, err
	}
	if record.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (s *Store) Create(ctx context.Context, userID string, in CreateInput) (Record, error) {
	if err := in.Validate(); err != nil {
		return Record{}, fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	now := s.nowFunc().Format(time.RFC3339)
	record := Record{
		ID: s.idFunc(), UserID: userID, Kind: in.Kind,
		Title: strings.TrimSpace(in.Title), Input: in.Input, Output: in.Output,
	}
	var output any
	if len(in.Output) > 0 {
		output = string(in.Output)
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO tool_records (id,user_id,kind,title,input,output,created_at,updated_at)
VALUES (?,?,?,?,?,?,?,?)`,
		record.ID, userID, string(record.Kind), record.Title, string(record.Input), output, now, now)
	if err != nil {
		return Record{}, err
	}
	return s.Get(ctx, userID, record.ID)
}

func (s *Store) Get(ctx context.Context, userID, recordID string) (Record, error) {
	record, err := scanRecord(s.db.QueryRowContext(ctx, `
SELECT id,user_id,kind,title,input,output,created_at,updated_at
FROM tool_records WHERE id=? AND user_id=? AND deleted_at IS NULL`, recordID, userID).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return Record{}, ErrNotFound
	}
	return record, err
}

// List returns the newest records of one kind. Records are small JSON
// snapshots, so the list carries input/output directly and no detail endpoint
// is needed.
func (s *Store) List(ctx context.Context, userID string, kind Kind, limit int) ([]Record, error) {
	if kind != "" && !kind.Valid() {
		return nil, fmt.Errorf("%w: unknown kind", ErrInvalid)
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := `
SELECT id,user_id,kind,title,input,output,created_at,updated_at
FROM tool_records WHERE user_id=? AND deleted_at IS NULL`
	args := []any{userID}
	if kind != "" {
		query += " AND kind=?"
		args = append(args, string(kind))
	}
	query += " ORDER BY updated_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]Record, 0)
	for rows.Next() {
		record, scanErr := scanRecord(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

// Rename updates the title only. updated_at is intentionally untouched:
// renaming is metadata, and the list orders by last activity.
func (s *Store) Rename(ctx context.Context, userID, recordID, title string) (Record, error) {
	title = strings.TrimSpace(title)
	if title == "" || utf8.RuneCountInString(title) > MaxTitleRunes {
		return Record{}, fmt.Errorf("%w: title must be 1-60 characters", ErrInvalid)
	}
	result, err := s.db.ExecContext(ctx, `
UPDATE tool_records SET title=? WHERE id=? AND user_id=? AND deleted_at IS NULL`, title, recordID, userID)
	if err != nil {
		return Record{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Record{}, err
	}
	if affected == 0 {
		return Record{}, ErrNotFound
	}
	return s.Get(ctx, userID, recordID)
}

func (s *Store) SoftDelete(ctx context.Context, userID, recordID string) error {
	result, err := s.db.ExecContext(ctx, `
UPDATE tool_records SET deleted_at=? WHERE id=? AND user_id=? AND deleted_at IS NULL`,
		s.nowFunc().Format(time.RFC3339), recordID, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
