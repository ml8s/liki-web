package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ml8s/liki-web/internal/adminauth"
	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/product"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

func TestAdminSessionReturnsCapabilitiesAndEntitlementRoutes(t *testing.T) {
	ctx := context.Background()
	authorizer, err := adminauth.Open(filepath.Join(t.TempDir(), "platform.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = authorizer.Close() })
	if err := authorizer.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := authorizer.BootstrapAdminByEmail(ctx, "support@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := authorizer.EnsureAdmin(ctx, "support@example.com", "reviewer@example.com", "create support reviewer"); err != nil {
		t.Fatal(err)
	}
	if err := authorizer.GrantCapability(ctx, "support@example.com", "reviewer@example.com", adminauth.CapabilityEntitlementRead, "support reads entitlements"); err != nil {
		t.Fatal(err)
	}
	if err := authorizer.GrantCapability(ctx, "support@example.com", "reviewer@example.com", adminauth.CapabilityEntitlementManage, "support updates entitlements"); err != nil {
		t.Fatal(err)
	}
	reviewer := auth.User{ID: "user-reviewer", Email: "reviewer@example.com", Status: auth.UserActive}

	entitlementDB, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = entitlementDB.Close() })
	entitlements := entitlement.NewStore(entitlementDB)
	if err := entitlements.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	granted, err := entitlements.GrantForProduct(ctx, "user-1", "customer@example.com", product.Chat, "test", "order-1", 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	RegisterRoutes(mux, nil, authorizer, func(*http.Request) (auth.User, bool) { return reviewer, true },
		nil, entitlements, nil, nil, nil, nil)
	execute := func(method, target, body string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(method, target, strings.NewReader(body))
		if body != "" {
			request.Header.Set("Content-Type", "application/json")
		}
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, request)
		return recorder
	}

	rec := execute(http.MethodGet, "/api/admin/session", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("session status=%d body=%s", rec.Code, rec.Body.String())
	}
	var session struct {
		Email        string   `json:"email"`
		Capabilities []string `json:"capabilities"`
	}
	var sessionEnvelope struct {
		Data struct {
			Email        string   `json:"email"`
			Capabilities []string `json:"capabilities"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &sessionEnvelope); err != nil {
		t.Fatal(err)
	}
	session = sessionEnvelope.Data
	if session.Email != reviewer.Email || len(session.Capabilities) != 2 {
		t.Fatalf("unexpected session: %+v", session)
	}

	rec = execute(http.MethodGet, "/api/admin/entitlements?email=customer@example.com", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("entitlements status=%d body=%s", rec.Code, rec.Body.String())
	}
	var list struct {
		Entitlements []entitlement.Entitlement `json:"entitlements"`
	}
	var listEnvelope struct {
		Data struct {
			Entitlements []entitlement.Entitlement `json:"entitlements"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listEnvelope); err != nil {
		t.Fatal(err)
	}
	list = listEnvelope.Data
	if len(list.Entitlements) != 1 || list.Entitlements[0].ID != granted.ID {
		t.Fatalf("unexpected entitlements: %+v", list.Entitlements)
	}

	expiresAt := time.Now().UTC().Add(72 * time.Hour).Format(time.RFC3339)
	rec = execute(http.MethodPatch, "/api/admin/entitlements/"+granted.ID+"/expire",
		`{"expires_at":"`+expiresAt+`","reason":"customer support extension"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expire status=%d body=%s", rec.Code, rec.Body.String())
	}
	updated, err := entitlements.ListForAdmin(ctx, "customer@example.com", "active", 10)
	if err != nil || len(updated) != 1 {
		t.Fatalf("updated entitlements: %+v err=%v", updated, err)
	}
	if !updated[0].ExpiresAt.Equal(expiresAtUTC(expiresAt)) {
		t.Fatalf("expires = %s, want %s", updated[0].ExpiresAt, expiresAt)
	}

	rec = execute(http.MethodPost, "/api/admin/entitlements/"+granted.ID+"/revoke", `{"reason":"duplicate purchase"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("revoke status=%d body=%s", rec.Code, rec.Body.String())
	}
	revoked, err := entitlements.ListForAdmin(ctx, "customer@example.com", "revoked", 10)
	if err != nil || len(revoked) != 1 || revoked[0].Status != entitlement.Revoked {
		t.Fatalf("revoked entitlements: %+v err=%v", revoked, err)
	}
}

func expiresAtUTC(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339, value)
	return parsed
}
