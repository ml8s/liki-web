package partner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminApplicationDecisionRequiresReason(t *testing.T) {
	store := newAdminStore(t)
	handler := NewAdminHandler(store, nil)
	request := httptest.NewRequest(http.MethodPost, "/api/admin/partner-applications/app/decision", strings.NewReader(`{"approve":true}`))
	request.SetPathValue("id", "app")
	recorder := httptest.NewRecorder()
	handler.HandleApplicationDecision()(recorder, request)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestAdminApplicationDecisionNotifiesAfterCommit(t *testing.T) {
	ctx := context.Background()
	store := newAdminStore(t)
	app, err := store.SubmitApplication(ctx, RoleScout, "applicant", "channel partner")
	if err != nil {
		t.Fatal(err)
	}
	called := false
	handler := NewAdminHandler(store, func(_ context.Context, item PartnerApplication, approved bool, reason string) {
		called = true
		if item.ID != app.ID || item.Type != RoleScout || !approved || reason != "qualified" {
			t.Fatalf("notification = %+v %v %q", item, approved, reason)
		}
	})
	request := httptest.NewRequest(http.MethodPost, "/decision", strings.NewReader(`{"approve":true,"reason":"qualified"}`))
	request.SetPathValue("id", app.ID)
	recorder := httptest.NewRecorder()
	handler.HandleApplicationDecision()(recorder, request)
	if recorder.Code != http.StatusOK || !called {
		t.Fatalf("status=%d called=%t body=%s", recorder.Code, called, recorder.Body.String())
	}
}

func TestAdminPartnerStatusRequiresReason(t *testing.T) {
	store := newAdminStore(t)
	handler := NewAdminHandler(store, nil)
	request := httptest.NewRequest(http.MethodPost, "/api/admin/partners/kol/freeze", strings.NewReader(`{}`))
	request.SetPathValue("userId", "kol")
	recorder := httptest.NewRecorder()
	handler.HandlePartnerFreeze()(recorder, request)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestAdminPartnerStatusFreezeAndUnfreeze(t *testing.T) {
	ctx := context.Background()
	store := newAdminStore(t)
	kol, err := store.createKOLPartner(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewAdminHandler(store, nil)
	request := httptest.NewRequest(http.MethodPost, "/api/admin/partners/kol/freeze", strings.NewReader(`{"reason":"manual review"}`))
	request.SetPathValue("userId", kol.UserID)
	recorder := httptest.NewRecorder()
	handler.HandlePartnerFreeze()(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("freeze status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	partner, err := store.ReferrerByUser(ctx, kol.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if partner.Status != PartnerFrozen {
		t.Fatalf("partner = %+v", partner)
	}
	unfreeze := httptest.NewRequest(http.MethodPost, "/api/admin/partners/kol/unfreeze", strings.NewReader(`{"reason":"verified"}`))
	unfreeze.SetPathValue("userId", kol.UserID)
	recorder = httptest.NewRecorder()
	handler.HandlePartnerUnfreeze()(recorder, unfreeze)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unfreeze status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}
