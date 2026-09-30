package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/ml8s/liki-web/internal/httputil"

	_ "modernc.org/sqlite"
)

func TestJWTAuth_NoCookie(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("GET", "/", nil)
	if _, _, ok := httputil.JWTAuth(r); ok {
		t.Error("expected no auth without cookie")
	}
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "liki_token", Value: "not-a-valid-jwt"})
	if _, _, ok := httputil.JWTAuth(r); ok {
		t.Error("expected no auth with invalid token")
	}
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	t.Parallel()
	claims := jwt.MapClaims{
		"email":    "user@example.com",
		"order_id": "order-1",
		"exp":      time.Now().Add(-time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(httputil.JWTSecret())
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "liki_token", Value: signed})
	if _, _, ok := httputil.JWTAuth(r); ok {
		t.Error("expected no auth with expired token")
	}
}

func TestJWTAuth_RoundTrip(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	if err := httputil.SetJWTCookie(w, "user@example.com", "order-1"); err != nil {
		t.Fatalf("SetJWTCookie failed: %v", err)
	}

	var tokenValue string
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "liki_token" {
			tokenValue = cookie.Value
			break
		}
	}
	if tokenValue == "" {
		t.Fatal("cookie not set")
	}

	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "liki_token", Value: tokenValue})
	email, orderID, ok := httputil.JWTAuth(r)
	if !ok {
		t.Fatal("jwtAuth failed")
	}
	if email != "user@example.com" || orderID != "order-1" {
		t.Fatalf("auth = %q/%q, want user@example.com/order-1", email, orderID)
	}
}

func TestJWTAuth_RejectsAlgorithmNone(t *testing.T) {
	t.Parallel()
	token := jwt.New(jwt.SigningMethodNone)
	token.Claims = jwt.MapClaims{
		"email":    "user@example.com",
		"order_id": "order-1",
		"exp":      time.Now().Add(time.Hour).Unix(),
	}
	signed, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none token: %v", err)
	}

	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "liki_token", Value: signed})
	if _, _, ok := httputil.JWTAuth(r); ok {
		t.Error("JWTAuth accepted token with algorithm=none")
	}
}
