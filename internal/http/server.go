package http

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"liki-web/internal/admin"
	"liki-web/internal/agent"
	"liki-web/internal/feedback"
	"liki-web/internal/hepan"
	"liki-web/internal/mingshu"
	"liki-web/internal/naming"
	"liki-web/internal/payment"
)

type ServerDeps struct {
	OnPaid         func(ctx context.Context, orderID string)
	OnReportReady  func(ctx context.Context, orderID string)
	OnVerification func(ctx context.Context, email, code string)
	Payment    *payment.Service
	NamingStore    *naming.Store
	MingShuStore    *mingshu.Store
	MingShuGenerator *mingshu.Generator
	HepanStore       *hepan.Store
	HepanGenerator    *hepan.Generator
	NamingAgent *agent.Agent
	Agent  *agent.Agent
	Analytics  *Analytics
	FeedbackStore   *feedback.Store
	FeedbackAPIKey  string
	DevMode    bool
}

func RegisterRoutes(mux *http.ServeMux, deps ServerDeps, buildTime, version string, rl *RateLimiter) http.Handler {
	// ── naming product routes ──
	if deps.NamingStore != nil {
		rateLimit := func(h http.HandlerFunc) http.HandlerFunc {
			return rl.Wrap(5.0/60, 6, h)
		}
		naming.RegisterRoutes(mux, deps.NamingStore, deps.Payment, deps.NamingAgent, deps.OnPaid, rateLimit)
	}

	mux.HandleFunc("GET /api/health", handleHealth())
	mux.HandleFunc("POST /api/analytics/pageview", handlePageView(deps.Analytics))
	mux.HandleFunc("GET /api/stats", handleStats(deps.Analytics))
	mux.HandleFunc("GET /api/version", handleVersion(buildTime, version))
	mux.HandleFunc("GET /api/location", handleLocation)

	// ── mingshu product routes ──
	if deps.MingShuStore != nil {
		mingshu.RegisterRoutes(mux, deps.MingShuStore, deps.Payment, deps.MingShuGenerator, deps.OnPaid, deps.OnReportReady)
	}

	// ── hepan product routes ──
	if deps.HepanStore != nil {
		hepan.RegisterRoutes(mux, deps.HepanStore, deps.Payment, deps.HepanGenerator, deps.OnPaid, deps.OnReportReady)
	}

	// ── admin routes (naming-specific — registered here to avoid import cycle) ──
	if deps.NamingStore != nil {
		admin.RegisterRoutes(mux, naming.NewAdminService(deps.NamingStore), deps.OnPaid)
	}
	// ── admin auth routes ──
	mux.HandleFunc("POST /api/admin/send-code", handleAdminSendCode(deps.OnVerification))
	mux.HandleFunc("POST /api/admin/verify-code", handleAdminVerifyCode())

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
		if distExists && serve(w, r, distDir) {
			return
		}
		if serve(w, r, webDir) {
			return
		}
		w.WriteHeader(http.StatusNotFound)
		serveFileNoRedirect(w, r, filepath.Join(webDir, "404.html"))
	})
}

func serveFileNoRedirect(w http.ResponseWriter, r *http.Request, path string) {
	f, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, filepath.Base(path), fi.ModTime(), f)
}
