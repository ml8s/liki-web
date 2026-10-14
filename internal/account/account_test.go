package account

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/birthprofile"
	"github.com/ml8s/liki-web/internal/chat"
	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/partner"
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/product"
	"github.com/ml8s/liki-web/internal/sales"
	"github.com/ml8s/liki-web/internal/sqliteutil"

	_ "modernc.org/sqlite"
)

func openTestDBNamed(t *testing.T, name string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	return openTestDBNamed(t, "app.db")
}

func TestHandleErasure(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	profileStore := birthprofile.NewStore(db)
	if err := profileStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	chatStore := chat.NewChatStore(db)
	if err := chatStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	orderStore := sales.NewOrderStoreWithCatalog(db, product.NewCatalog(product.CatalogConfig{
		OneTimePriceMinorCNY: 7900, OneTimePriceMinorUSD: 2990, SubscriptionPriceMinorUSD: 1990,
	}))
	entitlementStore := entitlement.NewStore(db)
	paymentStore := payment.NewIntentStore(db)
	notificationStore := payment.NewNotificationStore(db)
	referralDBPath := filepath.Join(t.TempDir(), "referral.db")
	referralDB, err := sqliteutil.Open(referralDBPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := referralDB.Close(); err != nil {
		t.Fatal(err)
	}
	referralStore, err := partner.Open(referralDBPath, []byte("test-referral-payout-secret"))
	if err != nil {
		t.Fatal(err)
	}
	if err := referralStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	authStore := auth.NewStore(db, []byte("test-auth-hmac"))
	if err := authStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := orderStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := entitlementStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := paymentStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := notificationStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	// Seed auth PII so the handler proves identity erasure, not only profiles.
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.ExecContext(ctx, `
INSERT INTO users (id,email,status,locale,created_at,updated_at)
VALUES ('user_1','u@example.com','active','zh-Hant',?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	authSeeds := []string{
		`INSERT INTO auth_identities (id,user_id,provider,provider_user_id,email,created_at) VALUES ('identity_1','user_1','email','u@example.com','u@example.com',?)`,
		`INSERT INTO auth_sessions (id,user_id,token_hash,status,ip_hash,user_agent,created_at,expires_at) VALUES ('session_1','user_1','hash','active','ip-hash','agent',?,?)`,
		`INSERT INTO auth_codes (id,email,code_hash,created_at,expires_at) VALUES ('code_1','u@example.com','hash',?,?)`,
	}
	for _, query := range authSeeds {
		if _, err := db.ExecContext(ctx, query, now, now); err != nil {
			t.Fatal(err)
		}
	}

	// user_1 建档案 + 会话 + system_profile 消息（含 PII）
	p, err := profileStore.Create(ctx, "user_1", birthprofile.BirthProfileInput{
		Nickname: "妈妈", Gender: "female", Calendar: "lunar",
		BirthDate: "1958-闰5-12", BirthTime: "06:00", BirthTimeKnown: "known",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := chatStore.CreateSession(ctx, "user_1", "liki", "标题", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := chatStore.AppendMessage(ctx, "user_1", session.ID, "assistant", chat.KindSystemProfile, `{"birth_date":"1958-闰5-12"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := chatStore.AppendMessage(ctx, "user_1", session.ID, "user", chat.KindText, "我出生于 1958 年闰五月十二，想问婚姻"); err != nil {
		t.Fatal(err)
	}

	handler := NewHandler(profileStore, chatStore, authStore,
		orderStore, entitlementStore, paymentStore, notificationStore,
		referralStore, nil, nil,
		func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				user := auth.User{ID: "user_1", Email: "u@example.com", Status: auth.UserActive}
				r = r.WithContext(auth.WithAuth(r.Context(), user, auth.AuthSession{ID: "s_1"}, "token"))
				next(w, r)
			}
		})

	req := httptest.NewRequest(http.MethodPost, "/api/account/erasure", nil)
	w := httptest.NewRecorder()
	handler.requireUser(handler.HandleErasure)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Status          string `json:"status"`
			ProfilesCleared int64  `json:"profiles_cleared"`
			MessagesCleared int64  `json:"messages_cleared"`

			AuthIdentitiesCleared int64 `json:"auth_identities_cleared"`
			AuthSessionsCleared   int64 `json:"auth_sessions_cleared"`
			AuthCodesCleared      int64 `json:"auth_codes_cleared"`

			ReferralFinanceTasks int64 `json:"referral_finance_tasks"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Data.Status != "erased" || resp.Data.ProfilesCleared != 1 || resp.Data.MessagesCleared != 2 ||
		resp.Data.AuthIdentitiesCleared != 1 || resp.Data.AuthSessionsCleared != 1 || resp.Data.AuthCodesCleared != 1 ||
		resp.Data.ReferralFinanceTasks != 0 {
		t.Fatalf("bad erasure result: %+v", resp.Data)
	}

	// 档案 PII 已清 + 软删；会话消息内容已清
	if _, err := profileStore.Get(ctx, "user_1", p.ID); err != birthprofile.ErrNotFound {
		t.Fatalf("profile should be gone, got err=%v", err)
	}
	msgs, err := chatStore.ListMessages(ctx, "user_1", session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 || strings.TrimSpace(msgs[0].Content) != "" || strings.TrimSpace(msgs[1].Content) != "" {
		t.Fatalf("message PII not cleared: %+v", msgs)
	}

	var email, status string
	if err := db.QueryRowContext(ctx, `SELECT email,status FROM users WHERE id='user_1'`).Scan(&email, &status); err != nil {
		t.Fatal(err)
	}
	var identities, sessions, codes int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM auth_identities WHERE user_id='user_1'`).Scan(&identities); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM auth_sessions WHERE user_id='user_1'`).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM auth_codes WHERE email='u@example.com'`).Scan(&codes); err != nil {
		t.Fatal(err)
	}
	if status != "deleted" || identities != 0 || sessions != 0 || codes != 0 || strings.Contains(email, "u@example.com") {
		t.Fatalf("auth tombstone email=%q status=%q identities=%d sessions=%d codes=%d", email, status, identities, sessions, codes)
	}
}
