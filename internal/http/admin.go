package http

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"liki-web/internal/httputil"
)

// ── admin-only middleware ──

// ── admin login (verification code) ──

// adminCodeStore holds in‑memory verification codes.
type adminCodeStore struct {
	mu    sync.Mutex
	codes map[string]adminCode
}

type adminCode struct {
	code      string
	expiresAt time.Time
}

var adminCodes = &adminCodeStore{codes: make(map[string]adminCode)}

func (s *adminCodeStore) set(email, code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[email] = adminCode{code: code, expiresAt: time.Now().Add(5 * time.Minute)}
	// Clean expired entries.
	for k, v := range s.codes {
		if time.Now().After(v.expiresAt) {
			delete(s.codes, k)
		}
	}
}

func (s *adminCodeStore) verify(email, code string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.codes[email]
	if !ok || time.Now().After(v.expiresAt) {
		delete(s.codes, email)
		return false
	}
	delete(s.codes, email) // one‑time use
	return v.code == code
}

type adminSendCodeRequest struct {
	Email string `json:"email"`
}

func handleAdminSendCode(onVerification func(ctx context.Context, email, code string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := decodeJSON[adminSendCodeRequest](w, r)
		if !ok {
			return
		}
		adminEmail := os.Getenv("ADMIN_EMAIL")
		if req.Email != adminEmail {
			respondJSON(w, http.StatusOK, map[string]string{"result": "ok"})
			return
		}

		code := fmt.Sprintf("%06d", rand.Intn(1000000)) //nolint:gosec
		adminCodes.set(req.Email, code)

		go func() {
			if onVerification != nil {
				onVerification(context.Background(), req.Email, code)
			} else {
				slog.Warn("admin: no verification callback set")
			}
		}()

		respondJSON(w, http.StatusOK, map[string]string{"result": "ok"})
	}
}

type adminVerifyCodeRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func handleAdminVerifyCode() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := decodeJSON[adminVerifyCodeRequest](w, r)
		if !ok {
			return
		}

		if !adminCodes.verify(req.Email, req.Code) {
			respondError(w, http.StatusForbidden, httputil.ErrCodeForbidden, "验证码错误或已过期")
			return
		}

		if httputil.SetJWTCookieOrRespondErr(w, req.Email, "") != nil {
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"result": "ok"})
	}
}

// ── order management ──
