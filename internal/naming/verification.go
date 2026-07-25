package naming

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

const codeCharset = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"
const codeLength = 8

func generateCode() (string, error) {
	n := big.NewInt(int64(len(codeCharset)))
	b := make([]byte, codeLength)
	for i := range b {
		idx, err := rand.Int(rand.Reader, n)
		if err != nil {
			return "", fmt.Errorf("generate code: %w", err)
		}
		b[i] = codeCharset[idx.Int64()]
	}
	return string(b), nil
}

// ── Store methods ──

func (s *Store) GenerateVerificationCode(ctx context.Context, createdBy string) (string, error) {
	for i := 0; i < 10; i++ {
		code, err := generateCode()
		if err != nil {
			return "", err
		}
		_, err = s.db.ExecContext(ctx,
			`INSERT INTO verification_codes (code, status, created_by) VALUES (?, 'active', ?)`,
			code, createdBy)
		if err == nil {
			return code, nil
		}
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			continue
		}
		return "", fmt.Errorf("naming: generate code: %w", err)
	}
	return "", errors.New("naming: failed to generate unique code after 10 attempts")
}

func (s *Store) BatchGenerateVerificationCodes(ctx context.Context, count int, createdBy string) ([]string, error) {
	codes := make([]string, count)
	for i := range codes {
		code, err := generateCode()
		if err != nil {
			return nil, err
		}
		codes[i] = code
	}

	for attempt := 0; attempt < 3; attempt++ {
		err := s.batchInsertCodes(ctx, codes, createdBy)
		if err == nil {
			return codes, nil
		}
		if !strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, fmt.Errorf("naming: batch generate: %w", err)
		}
		for i := range codes {
			code, err := generateCode()
			if err != nil {
				return nil, err
			}
			codes[i] = code
		}
	}
	return nil, errors.New("naming: failed to generate unique codes after 3 attempts")
}

func (s *Store) batchInsertCodes(ctx context.Context, codes []string, createdBy string) error {
	stmt := `INSERT INTO verification_codes (code, status, created_by) VALUES `
	args := make([]any, 0, len(codes)*3)
	for i, c := range codes {
		if i > 0 {
			stmt += ", "
		}
		stmt += "(?, 'active', ?)"
		args = append(args, c, createdBy)
	}
	_, err := s.db.ExecContext(ctx, stmt, args...)
	return err
}
