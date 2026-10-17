package http

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ml8s/liki-web/internal/account"
	"github.com/ml8s/liki-web/internal/admin"
	"github.com/ml8s/liki-web/internal/adminauth"
	"github.com/ml8s/liki-web/internal/aipay"
	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/birthprofile"
	"github.com/ml8s/liki-web/internal/chat"
	"github.com/ml8s/liki-web/internal/enginemcp"
	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/feedback"
	"github.com/ml8s/liki-web/internal/partner"
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/sales"
)

type ServerDeps struct {
	PaymentRuntime       *payment.Runtime
	AuthHandler          *auth.Handler
	PartnerHandler       *partner.Handler
	PartnerStore         *partner.Store
	SalesHandler         *sales.Handler
	EntitlementHandler   *entitlement.Handler
	ChatHandler          *chat.Handler
	EngineMCP            *enginemcp.Handler
	AipayHandler         *aipay.Handler
	ProfileHandler       *birthprofile.Handler
	AccountHandler       *account.Handler
	Entitlements         *entitlement.Store
	MetricsHandler       http.HandlerFunc
	Readiness            ReadinessChecker
	FeedbackStore        *feedback.Store
	FeedbackAPIKey       string
	AdminEmail           string
	AdminAuthorizer      *adminauth.Store
	AuthUsers            *auth.Store
	SalesStores          *sales.SalesStores
	CompensationNotifier admin.CompensationNotifier
	PartnerNotifier      partner.DecisionNotifier
	DevMode              bool
}

func RegisterRoutes(mux *http.ServeMux, deps ServerDeps, buildTime, version string, rl, verifyRL *RateLimiter) http.Handler {
	// 产品域共用限流档（5/min burst 6）：支付创建类端点
	rateLimit := func(h http.HandlerFunc) http.HandlerFunc {
		return rl.Wrap(5.0/60, 6, h)
	}

	// 支付回调限流档：webhook 来自固定签名方但可能集中到达，且需抑制
	// 未签名伪造请求的滥用；60/min burst 30 宽松到不误伤正常回调。
	webhookLimit := func(h http.HandlerFunc) http.HandlerFunc {
		return rl.Wrap(60.0/60, 30, h)
	}

	// ── unified payment runtime ──
	if deps.PaymentRuntime != nil {
		mux.HandleFunc("GET /api/payments/intents/{id}/status", deps.PaymentRuntime.HandleIntentStatus)
		mux.HandleFunc("GET /api/payments/intents/{id}/payment-completion-status", deps.PaymentRuntime.HandlePaymentCompletionStatus)
		mux.HandleFunc("GET /api/payments/return/{id}", deps.PaymentRuntime.HandlePaymentReturn)
		mux.HandleFunc("POST /api/payments/webhook", webhookLimit(deps.PaymentRuntime.HandleWebhook))
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

	// ── product / sales / entitlement / chat bounded contexts ──
	if deps.SalesHandler != nil {
		deps.SalesHandler.RegisterRoutes(mux)
	}
	if deps.EntitlementHandler != nil {
		deps.EntitlementHandler.RegisterRoutes(mux)
	}
	if deps.ChatHandler != nil {
		deps.ChatHandler.RegisterRoutes(mux)
	}
	if deps.EngineMCP != nil {
		engineLimit := func(next http.HandlerFunc) http.HandlerFunc {
			return rl.Wrap(30.0/60, 30, next)
		}
		deps.EngineMCP.RegisterRoutes(mux, engineLimit)
	}
	if deps.AipayHandler != nil {
		// Aipay bills and proof redemption are write-like endpoints and get
		// a stricter budget than ordinary free API reads.
		aipayLimit := func(next http.HandlerFunc) http.HandlerFunc {
			return rl.Wrap(2.0/60, 4, next)
		}
		deps.AipayHandler.RegisterRoutes(mux, aipayLimit)
	}

	// ── birth profiles ──
	if deps.ProfileHandler != nil {
		birthProfileLimit := func(next http.HandlerFunc) http.HandlerFunc {
			return rl.Wrap(10.0/60, 12, next)
		}
		deps.ProfileHandler.RegisterRoutes(mux, birthProfileLimit)
	}

	// ── partner roles（KOL / Scout 独立角色）──
	if deps.PartnerHandler != nil {
		deps.PartnerHandler.RegisterRoutes(mux, rateLimit)
	}

	// ── platform account（合规强删）──
	if deps.AccountHandler != nil {
		// Erasure spans every bounded context. Keep this expensive, destructive
		// operation on a much tighter budget than ordinary authenticated reads.
		erasureLimit := func(next http.HandlerFunc) http.HandlerFunc {
			return verifyRL.Wrap(1.0/600, 2, next)
		}
		deps.AccountHandler.RegisterRoutes(mux, erasureLimit)
	}

	mux.HandleFunc("GET /api/health", handleHealth())
	readinessLimit := func(next http.HandlerFunc) http.HandlerFunc {
		return rl.Wrap(10.0/60, 10, next)
	}
	mux.HandleFunc("GET /api/ready", readinessLimit(handleReady(deps.Readiness)))
	analyticsLimit := func(next http.HandlerFunc) http.HandlerFunc {
		return rl.Wrap(30.0/60, 30, next)
	}
	mux.HandleFunc("POST /api/analytics/pageview", analyticsLimit(handlePageView()))
	mux.HandleFunc("GET /api/version", handleVersion(buildTime, version))
	if deps.MetricsHandler != nil {
		rlInternal := func(next http.HandlerFunc) http.HandlerFunc {
			return rl.Wrap(0.5, 10, next)
		}
		mux.HandleFunc("GET /api/internal/metrics", rlInternal(deps.MetricsHandler))
	}

	// ── admin routes (registered here to avoid import cycle) ──
	if deps.PartnerStore != nil {
		if deps.AuthHandler != nil || deps.AdminEmail != "" {
			adminIdentity := func(r *http.Request) (auth.User, bool) {
				if deps.AuthHandler == nil {
					return auth.User{}, false
				}
				return deps.AuthHandler.IdentityFromRequest(r)
			}
			admin.RegisterRoutes(mux, deps.PartnerStore, deps.AdminAuthorizer, adminIdentity,
				deps.PartnerNotifier, deps.Entitlements, deps.AuthUsers, deps.SalesStores,
				deps.PartnerStore, deps.CompensationNotifier)
		}
	}
	// ── admin auth routes（限流：6 位验证码 5 分钟窗口，无限流即可暴力——1/min burst 3）──
	// send-code（防邮件轰炸）与 verify-code（防暴破）是独立风险面：独立限流桶，
	// 避免共享桶互相干扰（打 send 耗尽配额锁死 verify 的 DoS 放大 + 冒烟连发误伤）

	// ── feedback ──
	if deps.FeedbackStore != nil {
		rlFeedback := func(next http.HandlerFunc) http.HandlerFunc {
			return rl.Wrap(5.0/60, 1, next)
		}
		// 管理面（API key 保护）独立桶：读 30/min、写 10/min。
		rlFeedbackOps := func(next http.HandlerFunc) http.HandlerFunc {
			return rl.Wrap(0.5, 10, next)
		}
		mux.HandleFunc("POST /api/feedback", feedback.HandleCreate(deps.FeedbackStore, rlFeedback))
		mux.HandleFunc("GET /api/feedback", rlFeedbackOps(feedback.HandleList(deps.FeedbackStore, deps.FeedbackAPIKey)))
		mux.HandleFunc("PATCH /api/feedback/{id}", rlFeedbackOps(feedback.HandleUpdate(deps.FeedbackStore, deps.FeedbackAPIKey)))
	}

	handler := Recover(SecurityHeaders(SameOrigin(
		deps.DevMode,
		CORSMiddleware(deps.DevMode, BodyLimit(mux)),
	)))

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
		// Dynamic routes registered on the mux that are NOT under /api/ must be
		// listed here or the static file handler shadows them (currently: /r/{code}).
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/r/") {
			next.ServeHTTP(w, r)
			return
		}
		clean := filepath.Clean(r.URL.Path)
		if strings.HasPrefix(clean, "/app/assets/") {
			assetPath := filepath.Join(distDir, "app", strings.TrimPrefix(clean, "/app/"))
			if info, err := os.Stat(assetPath); err == nil && !info.IsDir() {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
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
