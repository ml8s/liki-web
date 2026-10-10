package httputil

import (
	"crypto/hkdf"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// capabilityMACKey derives an independent HMAC key for capability tokens via
// HKDF from the platform secret: sharing raw key material across the JWT
// signing context and this MAC context would violate key-separation
// practice. Computed once; sha256 never returns an error.
var capabilityMACKey = sync.OnceValue(func() []byte {
	key, err := hkdf.Key(sha256.New, JWTSecret(), nil, "liki capability-token v1", 32)
	if err != nil {
		panic(err)
	}
	return key
})

// NewCapabilityToken issues a bounded, action-specific capability token
// (RFC 7519 JWT, HS256 with the HKDF-derived key) for short-lived payment
// polling and other browser callbacks that should not rely on an
// unguessable resource ID alone.
func NewCapabilityToken(action, subject string, issuedAt time.Time, ttl time.Duration) (string, error) {
	if action == "" || subject == "" {
		return "", fmt.Errorf("token: action and subject are required")
	}
	if ttl <= 0 {
		return "", fmt.Errorf("token: ttl must be positive")
	}
	claims := jwt.RegisteredClaims{
		Audience:  jwt.ClaimStrings{action},
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(issuedAt),
		ExpiresAt: jwt.NewNumericDate(issuedAt.Add(ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(capabilityMACKey())
}

// VerifyCapabilityToken validates a capability token for exactly one action
// and subject: signature (HS256, constant-time via library), expiry, and
// audience must all match.
func VerifyCapabilityToken(action, subject, token string, now time.Time) bool {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithAudience(action),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	var claims jwt.RegisteredClaims
	if _, err := parser.ParseWithClaims(token, &claims, func(*jwt.Token) (any, error) {
		return capabilityMACKey(), nil
	}); err != nil {
		return false
	}
	return claims.Subject == subject
}
