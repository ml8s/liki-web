package http

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/naming"
	"github.com/ml8s/liki-web/internal/product"

	_ "modernc.org/sqlite"
)

func newAuthTestStore(t *testing.T) *naming.Store {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	return naming.NewStore(db)
}

func seedActiveOrder(t *testing.T, store *naming.Store, orderID, email string, prod product.Product, expiresAt string) {
	t.Helper()
	if err := store.CreateOrder(context.Background(), orderID, string(prod), 990, "CNY", "", "zh-Hans"); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := store.UpdateEmail(context.Background(), orderID, email); err != nil {
		t.Fatalf("update email: %v", err)
	}
	if _, err := store.MarkPaidIdempotent(context.Background(), orderID, "pay-"+orderID); err != nil {
		t.Fatalf("mark paid: %v", err)
	}
	if err := store.OverrideChatExpiresAt(context.Background(), orderID, expiresAt); err != nil {
		t.Fatalf("set expiry: %v", err)
	}
}

func TestRetiredNamingLoginRoutesFailClosed(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := naming.NewStore(db)

	tests := []struct {
		name    string
		target  string
		handler http.HandlerFunc
	}{
		{
			name:    "email login",
			target:  "/api/naming/auth/login",
			handler: naming.HandleLogin(store),
		},
		{
			name:    "order select",
			target:  "/api/naming/orders/select",
			handler: naming.HandleOrderSelect(store),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := httptest.NewRequest(http.MethodPost, test.target, strings.NewReader(`{}`))
			response := httptest.NewRecorder()
			test.handler(response, request)
			if response.Code != http.StatusGone {
				t.Fatalf("status = %d, want %d, body: %s", response.Code, http.StatusGone, response.Body.String())
			}
			if cookies := response.Result().Cookies(); len(cookies) != 0 {
				t.Fatalf("retired route set cookies: %#v", cookies)
			}
		})
	}
}

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
