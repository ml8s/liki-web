package httputil

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func JWTSecret() []byte {
	return []byte(os.Getenv("JWT_SECRET"))
}

func ValidateJWTSecret(secret string) error {
	if secret == "" {
		return fmt.Errorf("JWT_SECRET environment variable is required")
	}
	if len(secret) < 32 {
		return fmt.Errorf("JWT_SECRET must contain at least 32 bytes")
	}
	return nil
}

func JWTAuth(r *http.Request) (email, orderID string, ok bool) {
	cookie, err := r.Cookie("liki_token")
	if err != nil {
		return "", "", false
	}
	token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (any, error) {
		return JWTSecret(), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !token.Valid {
		return "", "", false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", false
	}
	email, _ = claims["email"].(string)
	orderID, _ = claims["order_id"].(string)
	return email, orderID, email != "" && orderID != ""
}

func SetJWTCookie(w http.ResponseWriter, email, orderID string) error {
	claims := jwt.MapClaims{
		"email":    email,
		"order_id": orderID,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(JWTSecret())
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "liki_token",
		Value:    signed,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})
	return nil
}

// SetJWTCookieOrRespondErr sets the JWT cookie and writes an error response on failure.
// Returns the error so callers can stop further processing.
func SetJWTCookieOrRespondErr(w http.ResponseWriter, email, orderID string) error {
	if err := SetJWTCookie(w, email, orderID); err != nil {
		slog.Error("auth: sign jwt", "err", err)
		RespondError(w, http.StatusInternalServerError, ErrCodeInternal, "登录失败")
		return err
	}
	return nil
}
