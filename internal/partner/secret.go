package partner

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// fieldSealer encrypts sensitive payout account values before SQLite persistence.
// It is intentionally separate from authentication: rotating JWT sessions must
// not make stored payout instructions unreadable, so the key is derived from the
// deployment secret with a dedicated domain label.
type fieldSealer struct {
	aead cipher.AEAD
}

func newFieldSealer(secret []byte) (*fieldSealer, error) {
	if len(secret) == 0 {
		return nil, errors.New("partner: payout field secret is required")
	}
	key := sha256.Sum256(append([]byte("liki:referral:payout-account:v1:"), secret...))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("partner: create payout account cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("partner: create payout account AEAD: %w", err)
	}
	return &fieldSealer{aead: aead}, nil
}

func (s *fieldSealer) seal(plaintext string) (string, error) {
	if plaintext == "" {
		return "", errors.New("partner: payout account is required")
	}
	if strings.HasPrefix(plaintext, "v1:") {
		return plaintext, nil
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("partner: generate payout account nonce: %w", err)
	}
	sealed := s.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return "v1:" + base64.RawStdEncoding.EncodeToString(sealed), nil
}

func (s *fieldSealer) open(value string) (string, error) {
	if value == "" {
		return "", errors.New("partner: payout account is empty")
	}
	encoded, ok := strings.CutPrefix(value, "v1:")
	if !ok {
		// Values migrated before field encryption are re-sealed by migration 009.
		return value, nil
	}
	raw, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("partner: decode payout account: %w", err)
	}
	size := s.aead.NonceSize()
	if len(raw) <= size {
		return "", errors.New("partner: invalid payout account ciphertext")
	}
	plaintext, err := s.aead.Open(nil, raw[:size], raw[size:], nil)
	if err != nil {
		return "", fmt.Errorf("partner: decrypt payout account: %w", err)
	}
	return string(plaintext), nil
}
