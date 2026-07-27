package main

import (
	"context"
	"encoding/json"
	"flag"
	"time"
	"liki-web/internal/product"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"liki-web/internal/agent"
	"liki-web/internal/hepan"
	"liki-web/internal/mingshu"
	"liki-web/internal/naming"
	"liki-web/internal/dodo"
	"liki-web/internal/xunhu"

	_ "modernc.org/sqlite"

	"liki-web/internal/email"
	"liki-web/internal/feedback"
	apphttp "liki-web/internal/http"
	"liki-web/internal/httputil"
	"liki-web/internal/llm"
	"liki-web/internal/payment"
)

// Build-time variables injected via -ldflags.
var (
	BuildTime = "dev"
	Version   = "dev"
)

type Config struct {
	DBPath          string
	LifebookDBPath  string
	HepanDBPath     string
	FeedbackDBPath  string
	FeedbackAPIKey  string
	ResendAPIKey    string
	ResendFrom      string
	DodoAPIKey      string
	DodoWebhookKey  string
	DodoTestMode    bool
	DodoProductNaming string
	XunhuAppID      string
	XunhuAppSecret  string
	DeepseekAPIKey  string
	LLMModel        string
	LLMThinking     bool
	SkillsDir       string
	PromptFile      string
	EngineURL       string
	ReturnURL       string
	JWTSecret       string
	ListenAddr      string
	DevMode         bool
}

func LoadConfig() Config {
	return Config{
		DBPath:            envOr("DB_PATH", "/var/lib/liki/naming.db"),
		LifebookDBPath:    envOr("MINGSHU_DB_PATH", "/var/lib/liki/mingshu.db"),
		HepanDBPath:       envOr("HEPAN_DB_PATH", "/var/lib/liki/hepan.db"),
		FeedbackDBPath:    envOr("FEEDBACK_DB_PATH", "/var/lib/liki/feedback.db"),
		FeedbackAPIKey:    os.Getenv("FEEDBACK_API_KEY"),
		ResendAPIKey:      os.Getenv("RESEND_API_KEY"),
		ResendFrom:        envOr("RESEND_FROM", envOr("EMAIL_FROM", "Liki <report@liki.email>")),
		DodoAPIKey:        envOr("DODO_API_KEY", ""),
		DodoWebhookKey:    envOr("DODO_WEBHOOK_KEY", ""),
		DodoTestMode:      envOrBool("DODO_TEST_MODE", false),
		DodoProductNaming: os.Getenv("DODO_PRODUCT_NAMING"),
		XunhuAppID:        envOr("XUNHU_APPID", ""),
		XunhuAppSecret:    envOr("XUNHU_APPSECRET", ""),
		DeepseekAPIKey:    envOr("DEEPSEEK_API_KEY", ""),
		LLMModel:          envOr("LLM_MODEL", ""),
		LLMThinking:       envOrBool("LLM_THINKING", false),
		SkillsDir:         envOr("LIKI_SKILLS_DIR", "web/skills"),
		PromptFile:        envOr("LIKI_PROMPT_FILE", "naming/SKILL.md"),
		EngineURL:         envOr("LIKI_ENGINE_URL", "http://localhost:8082/jsonrpc"),
		ReturnURL:         envOr("RETURN_URL", ""),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		ListenAddr:        envOr("LISTEN_ADDR", ":8080"),
		DevMode:           envOrBool("DEV_MODE", false),
	}
}

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dbPath := flag.String("db", "/var/lib/liki/naming.db", "SQLite database path")
	flag.Parse()

	cfg := LoadConfig()

	// Structured logging
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	// Validate required config
	requireEnv("JWT_SECRET", cfg.JWTSecret, "DEEPSEEK_API_KEY", cfg.DeepseekAPIKey, "LLM_MODEL", cfg.LLMModel)

	// Flags override env when explicitly set
	if *addr != ":8080" {
		cfg.ListenAddr = *addr
	}
	if *dbPath != "/var/lib/liki/naming.db" {
		cfg.DBPath = *dbPath
	}

	// Init database
	db, err := payment.OpenDB(cfg.DBPath)
	if err != nil {
		slog.Error("open db", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	namingStore := naming.NewStore(db)

	// ── feedback（共用 naming 数据库）──
	var feedbackStore *feedback.Store
	feedbackStore, err = feedback.NewStore(db)
	if err != nil {
		slog.Error("init feedback store", "err", err)
		os.Exit(1)
	}

	mingshuDB, err := payment.OpenDB(cfg.LifebookDBPath)
	if err != nil {
		slog.Error("init mingshu db", "err", err)
		os.Exit(1)
	}
	mingshuStore := mingshu.NewStore(mingshuDB)

	// Init services
	emailClient := email.New(cfg.ResendAPIKey, cfg.ResendFrom)

	dodoProducts := map[product.Product]string{
		product.ProductNaming: cfg.DodoProductNaming,
	}
	dodoClient := dodo.New(cfg.DodoAPIKey, cfg.DodoWebhookKey, cfg.DodoTestMode, dodoProducts)

	xunhuClient := xunhu.New(cfg.XunhuAppID, cfg.XunhuAppSecret)

	llmClient := llm.New(cfg.DeepseekAPIKey, cfg.LLMModel, cfg.LLMThinking)
	prompt, err := os.ReadFile(filepath.Join(cfg.SkillsDir, cfg.PromptFile))
	if err != nil {
		slog.Error("read prompt file", "err", err, "dir", cfg.SkillsDir, "file", cfg.PromptFile)
		os.Exit(1)
	}

	tools := agent.NewToolRegistry(cfg.EngineURL)
	tools.Register("read_file",
		"Read a file from the skills directory. Use this to load skill instructions or report templates.",
		json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Relative path within skills directory, e.g. qiming/SKILL.md"}},"required":["path"]}`),
		agent.ReadFileHandler(cfg.SkillsDir))

	chatAgent := agent.NewAgent(llmClient, tools, string(prompt))
	mingshuGenerator := mingshu.NewGenerator(chatAgent, mingshuStore)
	if err := mingshuGenerator.LoadPromptsFrom(filepath.Join(cfg.SkillsDir, "reports/mingshu")); err != nil {
		slog.Error("mingshu: load prompts", "err", err)
		os.Exit(1)
	}

	// ── hepan ──
	hepanDB, err := payment.OpenDB(cfg.HepanDBPath)
	if err != nil {
		slog.Error("open hepan db", "err", err)
		os.Exit(1)
	}
	defer hepanDB.Close()
	if _, err := hepanDB.Exec(hepan.Schema); err != nil {
		slog.Error("init hepan schema", "err", err)
		os.Exit(1)
	}
	hepanStore := hepan.NewStore(hepanDB)
	hepanGenerator := hepan.NewGenerator(chatAgent, hepanStore)
	if err := hepanGenerator.LoadPromptsFrom(filepath.Join(cfg.SkillsDir, "reports/hepan")); err != nil {
		slog.Error("load hepan prompts", "err", err)
		os.Exit(1)
	}

	if cfg.ReturnURL == "" {
		slog.Warn("RETURN_URL not set — post-payment redirect will fail")
	}

	// Context for background tasks and server BaseContext.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── Callbacks ──
	onPaid := namingOnPaid(namingStore, emailClient, cfg.ReturnURL)
	onMingReady := mingshuOnReportReady(mingshuStore, emailClient, cfg.ReturnURL)
	onHepanReady := hepanOnReportReady(hepanStore, emailClient, cfg.ReturnURL)
	onVerification := adminVerification(emailClient)

	paymentSvc := payment.NewService(dodoClient, xunhuClient, cfg.ReturnURL)

	// Validate JWT_SECRET at startup so config errors fail fast.
	if err := httputil.ValidateJWTSecret(cfg.JWTSecret); err != nil {
		slog.Error("validate JWT_SECRET", "err", err)
		os.Exit(1)
	}

	// Setup HTTP
	rateLimiter := apphttp.NewRateLimiter()
	defer rateLimiter.Stop()

	mux := http.NewServeMux()
	h := apphttp.RegisterRoutes(mux, apphttp.ServerDeps{
		OnPaid:         onPaid,
		OnReportReady:  func(ctx context.Context, orderID string) {
			onMingReady(ctx, orderID)
			onHepanReady(ctx, orderID)
		},
		OnVerification: onVerification,
		Payment:   paymentSvc,
		NamingStore:    namingStore,
		MingShuStore:    mingshuStore,
		MingShuGenerator: mingshuGenerator,
		HepanStore:       hepanStore,
		HepanGenerator:    hepanGenerator,
		FeedbackStore:   feedbackStore,
		FeedbackAPIKey:  cfg.FeedbackAPIKey,
		Agent: chatAgent,
		Analytics: apphttp.NewAnalytics(),
		DevMode:   cfg.DevMode,
	}, BuildTime, Version, rateLimiter)

	// Write timeout must accommodate LLM calls (~120s)
	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      h,
		BaseContext:  func(_ net.Listener) context.Context { return ctx },
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 150 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Background stale order cleanup
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in cleanupStaleOrders", "panic", r)
			}
		}()
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if n, err := namingStore.CleanStale(ctx); err != nil {
					slog.Error("clean stale orders", "err", err)
				} else if n > 0 {
					slog.Info("cleaned stale orders", "count", n)
				}
			}
		}
	}()

	// Signal handling
	go handleSignals(srv)

	slog.Info("liki listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
	slog.Info("server stopped")

	}

func requireEnv(pairs ...string) {
	for i := 0; i < len(pairs); i += 2 {
		if pairs[i+1] == "" {
			slog.Error("required environment variable not set", "var", pairs[i])
			os.Exit(1)
		}
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envOrBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			slog.Warn("invalid bool env", "key", key, "value", v, "default", def)
			return def
		}
		return b
	}
	return def
}



func handleSignals(srv *http.Server) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic in handleSignals", "panic", r)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("received signal, shutting down", "signal", sig.String())
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("forced shutdown", "err", err)
	}
}
