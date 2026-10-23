package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ml8s/liki-web/internal/adminauth"
	"github.com/ml8s/liki-web/internal/auth"
)

func TestCapabilityGuard_RequiresUnifiedSession(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "platform.db")
	authorizer, err := adminauth.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = authorizer.Close() })
	if err := authorizer.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	identity := func(*http.Request) (auth.User, bool) { return auth.User{}, false }
	guard := capabilityGuard(authorizer, identity)(adminauth.CapabilityPartnerRead)(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler was called")
	})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/session", nil)
	rec := httptest.NewRecorder()
	guard(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestCapabilityGuard_RequiresCapability(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "platform.db")
	authorizer, err := adminauth.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = authorizer.Close() })
	if err := authorizer.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	identity := func(*http.Request) (auth.User, bool) {
		return auth.User{ID: "u1", Email: "u@example.com", Status: auth.UserActive}, true
	}
	guard := capabilityGuard(authorizer, identity)(adminauth.CapabilityPartnerApprove)(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler was called")
	})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/session", nil)
	req.AddCookie(&http.Cookie{Name: "liki_session", Value: "unified-session"})
	rec := httptest.NewRecorder()
	guard(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestAdminManagementRoutes(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "platform.db")
	authorizer, err := adminauth.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = authorizer.Close() })
	if err := authorizer.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := authorizer.BootstrapAdminByEmail(ctx, "owner@example.com"); err != nil {
		t.Fatal(err)
	}
	owner := auth.User{ID: "user-owner", Email: "owner@example.com", Status: auth.UserActive}
	identity := func(*http.Request) (auth.User, bool) { return owner, true }

	mux := http.NewServeMux()
	RegisterRoutes(mux, nil, authorizer, identity, nil, nil, nil, nil, nil, nil)

	execute := func(method, target, body string) *httptest.ResponseRecorder {
		t.Helper()
		var reader *strings.Reader
		if body == "" {
			reader = strings.NewReader("")
		} else {
			reader = strings.NewReader(body)
		}
		request := httptest.NewRequest(method, target, reader)
		if body != "" {
			request.Header.Set("Content-Type", "application/json")
		}
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, request)
		return recorder
	}

	if rec := execute(http.MethodGet, "/api/admin/admins", ""); rec.Code != http.StatusOK {
		t.Fatalf("list admins status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec := execute(http.MethodPost, "/api/admin/admins", `{"email":"ops@example.com"}`); rec.Code != http.StatusCreated {
		t.Fatalf("create admin status=%d body=%s", rec.Code, rec.Body.String())
	}
	rec := execute(http.MethodPost, "/api/admin/admins/ops@example.com/capabilities",
		`{"capability":"payout.read","reason":"finance reviewer"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("grant status=%d body=%s", rec.Code, rec.Body.String())
	}

	admins, err := authorizer.ListAdmins(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, account := range admins {
		if account.Email == "ops@example.com" && len(account.Capabilities) == 1 &&
			account.Capabilities[0].Capability == adminauth.CapabilityPayoutRead {
			found = true
		}
	}
	if !found {
		t.Fatalf("granted admin missing capability: %+v", admins)
	}

	rec = execute(http.MethodPost, "/api/admin/admins/owner@example.com/capabilities",
		`{"capability":"admin.super","reason":"self rotation"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("self management status=%d body=%s", rec.Code, rec.Body.String())
	}
}
