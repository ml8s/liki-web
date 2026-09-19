package http

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"liki-web/internal/account"
	"liki-web/internal/admin"
	"liki-web/internal/agent"
	"liki-web/internal/aipay"
	"liki-web/internal/auth"
	"liki-web/internal/chat"
	"liki-web/internal/commerce"
	"liki-web/internal/entitlement"
	"liki-web/internal/feedback"
	"liki-web/internal/hepan"
	"liki-web/internal/mingshu"
	"liki-web/internal/naming"
	"liki-web/internal/payment"
	"liki-web/internal/profile"
)

type ServerDeps struct {
	OnPaid             func(ctx context.Context, orderID string)
	OnVerification     func(ctx context.Context, email, code string)
	PaymentRuntime     *payment.Runtime
	AuthHandler        *auth.Handler
	EntitlementHandler *entitlement.Handler
	ChatHandler        *chat.Handler
	CommerceHandler    *commerce.Handler
	AipayHandler       *aipay.Handler
	ProfileHandler     *profile.Handler
	AccountHandler     *account.Handler
	NamingStore        *naming.Store
	MingShuStore       *mingshu.Store
	MingShuGenerator   *mingshu.Generator
	HepanStore         *hepan.Store
	HepanGenerator     *hepan.Generator
	NamingAgent        *agent.Agent
	Agent              *agent.Agent
	Analytics          *Analytics
	MetricsHandler     http.HandlerFunc
	FeedbackStore      *feedback.Store
	FeedbackAPIKey     string
	DevMode            bool
}

func RegisterRoutes(mux *http.ServeMux, deps ServerDeps, buildTime, version string, rl, verifyRL *RateLimiter) http.Handler {
	// 产品域共用限流档（5/min burst 6）：支付创建类端点
	rateLimit := func(h http.HandlerFunc) http.HandlerFunc {
		return rl.Wrap(5.0/60, 6, h)
	}

	// ── unified payment runtime ──
	if deps.PaymentRuntime != nil {
		mux.HandleFunc("POST /api/payments/intents", rateLimit(deps.PaymentRuntime.HandleCreateIntent))
		mux.HandleFunc("POST /api/payments/intents/{id}/checkout", rateLimit(deps.PaymentRuntime.HandleStartCheckout))
		mux.HandleFunc("GET /api/payments/intents/{id}/status", deps.PaymentRuntime.HandleIntentStatus)
		mux.HandleFunc("GET /api/payments/intents/{id}/fulfillment-status", deps.PaymentRuntime.HandleFulfillmentStatus)
		mux.HandleFunc("GET /api/payments/return/{id}", deps.PaymentRuntime.HandlePaymentReturn)
		mux.HandleFunc("POST /api/payments/webhook", deps.PaymentRuntime.HandleWebhook)
	}

	// ── platform user auth ──
	if deps.AuthHandler != nil {
		authCodeLimit := func(next http.HandlerFunc) http.HandlerFunc {
			return verifyRL.Wrap(1.0/60, 3, next)
		}
		authVerifyLimit := func(next http.HandlerFunc) http.HandlerFunc {
			return verifyRL.Wrap(5.0/60, 6, next)
		}
		deps.AuthHandler.RegisterRoutes(mux, authCodeLimit, authVerifyLimit)
	}

	// ── platform entitlements and unified chat ──
	if deps.EntitlementHandler != nil {
		deps.EntitlementHandler.RegisterRoutes(mux)
	}
	if deps.ChatHandler != nil {
		chatLimit := func(next http.HandlerFunc) http.HandlerFunc {
			return rl.Wrap(5.0/60, 6, next)
		}
		deps.ChatHandler.RegisterRoutes(mux, chatLimit)
	}
	if deps.CommerceHandler != nil {
		commerceLimit := func(next http.HandlerFunc) http.HandlerFunc {
			return rl.Wrap(5.0/60, 6, next)
		}
		deps.CommerceHandler.RegisterRoutes(mux, commerceLimit)
	}
	if deps.AipayHandler != nil {
		// Aipay bills and proof redemption are write-like endpoints and get
		// a stricter budget than ordinary free API reads.
		aipayLimit := func(next http.HandlerFunc) http.HandlerFunc {
			return rl.Wrap(2.0/60, 4, next)
		}
		deps.AipayHandler.RegisterRoutes(mux, aipayLimit)
	}

	// ── platform profiles ──
	if deps.ProfileHandler != nil {
		profileLimit := func(next http.HandlerFunc) http.HandlerFunc {
			return rl.Wrap(10.0/60, 12, next)
		}
		deps.ProfileHandler.RegisterRoutes(mux, profileLimit)
	}

	// ── platform account（合规强删）──
	if deps.AccountHandler != nil {
		deps.AccountHandler.RegisterRoutes(mux)
	}

	// ── naming product routes ──
	if deps.NamingStore != nil {
		naming.RegisterRoutes(mux, deps.NamingStore, deps.NamingAgent, rateLimit)
	}

	mux.HandleFunc("GET /api/health", handleHealth())
	mux.HandleFunc("POST /api/analytics/pageview", handlePageView(deps.Analytics))
	mux.HandleFunc("GET /api/stats", handleStats(deps.Analytics))
	mux.HandleFunc("GET /api/version", handleVersion(buildTime, version))
	mux.HandleFunc("GET /api/location", handleLocation)
	if deps.MetricsHandler != nil {
		mux.HandleFunc("GET /api/internal/metrics", deps.MetricsHandler)
	}

	// ── mingshu product routes ──
	if deps.MingShuStore != nil {
		mingshu.RegisterRoutes(mux, deps.MingShuStore, rateLimit)
	}

	// ── hepan product routes ──
	if deps.HepanStore != nil {
		hepan.RegisterRoutes(mux, deps.HepanStore, rateLimit)
	}

	// ── admin routes (naming-specific — registered here to avoid import cycle) ──
	if deps.NamingStore != nil {
		admin.RegisterRoutes(mux, naming.NewAdminService(deps.NamingStore), deps.OnPaid)
	}
	// ── admin auth routes（限流：6 位验证码 5 分钟窗口，无限流即可暴力——1/min burst 3）──
	// send-code（防邮件轰炸）与 verify-code（防暴破）是独立风险面：独立限流桶，
	// 避免共享桶互相干扰（打 send 耗尽配额锁死 verify 的 DoS 放大 + 冒烟连发误伤）
	mux.HandleFunc("POST /api/admin/send-code", rl.Wrap(1.0/60, 3, handleAdminSendCode(deps.OnVerification)))
	mux.HandleFunc("POST /api/admin/verify-code", verifyRL.Wrap(1.0/60, 3, handleAdminVerifyCode()))

	// ── feedback ──
	if deps.FeedbackStore != nil {
		rlFeedback := func(next http.HandlerFunc) http.HandlerFunc {
			return rl.Wrap(5.0/60, 1, next)
		}
		mux.HandleFunc("POST /api/feedback", feedback.HandleCreate(deps.FeedbackStore, rlFeedback))
		mux.HandleFunc("GET /api/feedback", feedback.HandleList(deps.FeedbackStore, deps.FeedbackAPIKey))
		mux.HandleFunc("PATCH /api/feedback/{id}", feedback.HandleUpdate(deps.FeedbackStore, deps.FeedbackAPIKey))
	}

	handler := Recover(SecurityHeaders(CORSMiddleware(deps.DevMode, BodyLimit(mux))))

	webDir := findWebDir()
	if webDir != "" {
		handler = fileServer(webDir, handler)
	}

	return handler
}

func findWebDir() string {
	for _, d := range []string{"web", "../web", "../../web"} {
		if fi, err := os.Stat(d); err == nil && fi.IsDir() {
			if abs, err := filepath.Abs(d); err == nil {
				return abs
			}
		}
	}
	return ""
}

func fileServer(webDir string, next http.Handler) http.Handler {
	distDir := filepath.Join(webDir, "dist")
	_, distErr := os.Stat(distDir)
	distExists := distErr == nil

	serve := func(w http.ResponseWriter, r *http.Request, root string) bool {
		clean := filepath.Clean(r.URL.Path)
		p := filepath.Join(root, clean[1:])

		if info, err := os.Stat(p); err == nil {
			if info.IsDir() {
				serveFileNoRedirect(w, r, filepath.Join(p, "index.html"))
				return true
			}
			http.ServeFile(w, r, p)
			return true
		}
		if _, err := os.Stat(p + ".html"); err == nil {
			serveFileNoRedirect(w, r, p+".html")
			return true
		}
		for _, loc := range []string{"/zh-Hans", "/zh-Hant", "/en"} {
			if strings.HasPrefix(clean, loc+"/") || clean == loc {
				serveFileNoRedirect(w, r, filepath.Join(root, loc[1:], "index.html"))
				return true
			}
		}
		defLoc := "zh-Hant"
		defP := filepath.Join(root, defLoc, clean)
		if info, err := os.Stat(defP); err == nil {
			if info.IsDir() {
				serveFileNoRedirect(w, r, filepath.Join(defP, "index.html"))
				return true
			}
			http.ServeFile(w, r, defP)
			return true
		}
		if _, err := os.Stat(defP + ".html"); err == nil {
			serveFileNoRedirect(w, r, defP+".html")
			return true
		}
		return false
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/jsonrpc") {
			next.ServeHTTP(w, r)
			return
		}
		clean := filepath.Clean(r.URL.Path)
		if strings.HasPrefix(clean, "/app/assets/") {
			assetPath := filepath.Join(distDir, "app", strings.TrimPrefix(clean, "/app/"))
			if info, err := os.Stat(assetPath); err == nil && !info.IsDir() {
				http.ServeFile(w, r, assetPath)
				return
			}
			http.NotFound(w, r)
			return
		}
		if clean == "/app" || strings.HasPrefix(clean, "/app/") {
			appIndex := filepath.Join(distDir, "app", "index.html")
			if _, err := os.Stat(appIndex); err == nil {
				serveFileNoRedirect(w, r, appIndex)
				return
			}
		}
		if distExists && serve(w, r, distDir) {
			return
		}
		if serve(w, r, webDir) {
			return
		}
		serveFileStatus(w, r, http.StatusNotFound, filepath.Join(webDir, "404.html"))
	})
}

func serveFileNoRedirect(w http.ResponseWriter, r *http.Request, path string) {
	f, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = f.Close() }()
	fi, err := f.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, filepath.Base(path), fi.ModTime(), f)
}

func serveFileStatus(w http.ResponseWriter, r *http.Request, status int, path string) {
	f, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = f.Close() }()
	fi, err := f.Stat()
	if err != nil || fi.IsDir() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.Copy(w, f)
}
