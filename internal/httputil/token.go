package httputil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const orderAccessTokenPrefix = "v1"

// NewCapabilityToken creates a bounded, action-specific capability. Use it for
// short-lived payment polling and other browser callbacks that should not rely
// on an unguessable resource ID alone.
func NewCapabilityToken(action, subject string, issuedAt time.Time, ttl time.Duration) (string, error) {
	if action == "" || subject == "" {
		return "", fmt.Errorf("token: action and subject are required")
	}
	if ttl <= 0 {
		return "", fmt.Errorf("token: ttl must be positive")
	}
	expiresAt := issuedAt.Add(ttl).Unix()
	payload := capabilityTokenPayload(action, subject, expiresAt)
	mac := hmac.New(sha256.New, JWTSecret())
	mac.Write([]byte(payload))
	return fmt.Sprintf("%s.%d.%s", orderAccessTokenPrefix, expiresAt, hex.EncodeToString(mac.Sum(nil))), nil
}

func VerifyCapabilityToken(action, subject, token string, now time.Time) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != orderAccessTokenPrefix {
		return false
	}
	expiresAt, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || now.Unix() >= expiresAt {
		return false
	}
	payload := capabilityTokenPayload(action, subject, expiresAt)
	mac := hmac.New(sha256.New, JWTSecret())
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(parts[2]))
}

func capabilityTokenPayload(action, subject string, expiresAt int64) string {
	return "capability:v1|" + action + "|" + subject + "|" + strconv.FormatInt(expiresAt, 10)
}
