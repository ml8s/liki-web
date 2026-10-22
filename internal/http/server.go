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
	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/feedback"
	"github.com/ml8s/liki-web/internal/partner"
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/records"
	"github.com/ml8s/liki-web/internal/runcontrol"
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
	RecordsHandler       *records.Handler
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
	SessionCookieName    string
	RunCallbackHandler   http.HandlerFunc
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

	// 支付自助读取限流档（capability-URL 类公开端点：返回/状态查询）。
	// 凭证是 intent id + 1h 作用域 token（见 ARCHITECTURE.md）；返回端点会触发
	// 一次 provider 对账调用，故即使 id 不可枚举也必须限流防匿名放大。
	// 60/min burst 30 对浏览器返回与状态轮询足够宽松。
	paymentSelfLimit := func(h http.HandlerFunc) http.HandlerFunc {
		return rl.Wrap(60.0/60, 30, h)
	}

	// ── unified payment runtime ──
	if deps.PaymentRuntime != nil {
		mux.HandleFunc("GET /api/payments/intents/{id}/status", paymentSelfLimit(deps.PaymentRuntime.HandleIntentStatus))
		mux.HandleFunc("GET /api/payments/intents/{id}/payment-completion-status", paymentSelfLimit(deps.PaymentRuntime.HandlePaymentCompletionStatus))
		mux.HandleFunc("GET /api/payments/return/{id}", paymentSelfLimit(deps.PaymentRuntime.HandlePaymentReturn))
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
	if deps.RecordsHandler != nil {
		deps.RecordsHandler.RegisterRoutes(mux)
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
	// 内部面共用限流档：边缘 forward_auth 与抓取都属低频小突发调用。
	rlInternal := func(next http.HandlerFunc) http.HandlerFunc {
		return rl.Wrap(0.5, 10, next)
	}
	if deps.MetricsHandler != nil {
		mux.HandleFunc("GET /api/internal/metrics", rlInternal(deps.MetricsHandler))
	}
	if deps.AuthUsers != nil && deps.Entitlements != nil && deps.SessionCookieName != "" {
		introspect := runcontrol.NewIntrospectHandler(runcontrol.IntrospectConfig{
			CookieName:   deps.SessionCookieName,
			Sessions:     deps.AuthUsers,
			Entitlements: deps.Entitlements,
		})
		// forward_auth 转发原始方法（浏览器对 /rt/<profile>/ag-ui 发 POST），
		// 因此这里按任意方法注册。
		mux.HandleFunc("/api/internal/introspect", rlInternal(introspect))
	}
	if deps.RunCallbackHandler != nil {
		mux.HandleFunc("POST /api/internal/runs/completed", rlInternal(deps.RunCallbackHandler))
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

	// API 链：Metrics 记录应用路由（静态文件不进入）；Recover 的 500 也被计入。
	handler := Metrics(Recover(SecurityHeaders(SameOrigin(
		deps.DevMode,
		CORSMiddleware(deps.DevMode, BodyLimit(mux)),
	))))

	webDir := findWebDir()
	if webDir != "" {
		handler = fileServer(webDir, handler)
	}
	// RequestID 最外层：静态页与 API 响应都带 X-Request-Id，日志与响应头一致。
	return RequestID(handler)
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

// marketingLocales 是营销站的语言前缀。
var marketingLocales = []string{"/zh-Hans", "/zh-Hant", "/en"}

// legacyPageTargets 记录已合并/下线的营销页（v5：bazi/mingshu 并入 natal；
// partners 下线为定向邀请制；hepan 已恢复独立页）。空目标表示落到语言首页。
var legacyPageTargets = map[string]string{"bazi": "natal", "mingshu": "natal", "partners": ""}

// legacyPageRedirect 返回下线页的 301 目标（/{locale}/{old}/ → /{locale}/{target}/ 或语言首页）。
func legacyPageRedirect(clean string) (string, bool) {
	for _, loc := range marketingLocales {
		if !strings.HasPrefix(clean, loc+"/") {
			continue
		}
		page := strings.TrimPrefix(clean, loc+"/")
		if target, ok := legacyPageTargets[page]; ok {
			if target == "" {
				return loc + "/", true
			}
			return loc + "/" + target + "/", true
		}
	}
	return "", false
}

// marketingPathLocale 返回路径的营销语言前缀；无前缀返回 false。
func marketingPathLocale(clean string) (string, bool) {
	for _, loc := range marketingLocales {
		if clean == loc || strings.HasPrefix(clean, loc+"/") {
			return loc, true
		}
	}
	return "", false
}

// negotiateLocale 按 Accept-Language 协商营销语言，与裸路径重定向契约一致。
func negotiateLocale(r *http.Request) string {
	accept := strings.ToLower(r.Header.Get("Accept-Language"))
	switch {
	case strings.HasPrefix(accept, "zh-cn"), strings.HasPrefix(accept, "zh-sg"), strings.HasPrefix(accept, "zh-hans"):
		return "zh-Hans"
	case strings.HasPrefix(accept, "zh-tw"), strings.HasPrefix(accept, "zh-hk"), strings.HasPrefix(accept, "zh-mo"), strings.HasPrefix(accept, "zh-hant"):
		return "zh-Hant"
	case strings.HasPrefix(accept, "en"):
		return "en"
	default:
		return "zh-Hant"
	}
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
				if !strings.HasSuffix(r.URL.Path, "/") {
					// 目录 URL 规范化：canonical 与 sitemap 均为带斜杠版本。
					http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
					return true
				}
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
		if _, ok := marketingPathLocale(clean); ok {
			// 旧 .html 营销 URL（bond 为合盘旧名）：目标存在时 301 到规范路径。
			if strings.HasSuffix(clean, ".html") {
				base := strings.TrimSuffix(clean, ".html")
				if strings.HasSuffix(base, "/bond") {
					// bond 为合盘旧名；合盘页现为 hepan。
					base = strings.TrimSuffix(base, "bond") + "hepan"
				}
				target := base + "/"
				if _, ok := marketingPathLocale(target); ok {
					if _, err := os.Stat(filepath.Join(root, target[1:], "index.html")); err == nil {
						http.Redirect(w, r, target, http.StatusMovedPermanently)
						return true
					}
				}
			}
			// 已合并页面（命理三页 → natal）：301 到新页，保留收取的权重。
			if target, ok := legacyPageRedirect(clean); ok {
				http.Redirect(w, r, target, http.StatusMovedPermanently)
				return true
			}
			// 不存在的营销路径必须 404：回落到语言首页会造成软 404 与索引稀释。
			return false
		}
		// 无语言前缀但存在于默认语言（zh-Hant）的路径：按 Accept-Language 协商
		// 301 到带前缀 URL；协商语言无此页面时回落默认语言。
		defP := filepath.Join(root, "zh-Hant", clean[1:])
		if info, err := os.Stat(defP); err == nil {
			loc := negotiateLocale(r)
			if _, err := os.Stat(filepath.Join(root, loc, clean[1:])); err != nil {
				loc = "zh-Hant"
			}
			target := "/" + loc + clean
			if info.IsDir() && !strings.HasSuffix(target, "/") {
				target += "/"
			}
			http.Redirect(w, r, target, http.StatusMovedPermanently)
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
		// 裸入口协商（与静态部署契约一致）：/, /skills → 本地化页面；
		// /skills/liki/... 子路径保持直接访问（agent artifact）。
		if clean == "/" {
			http.Redirect(w, r, "/"+negotiateLocale(r)+"/", http.StatusFound)
			return
		}
		if clean == "/skills" || clean == "/skills/" {
			http.Redirect(w, r, "/"+negotiateLocale(r)+"/skills/", http.StatusFound)
			return
		}
		if clean == "/admin.html" {
			http.Redirect(w, r, "/app/admin", http.StatusMovedPermanently)
			return
		}
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
