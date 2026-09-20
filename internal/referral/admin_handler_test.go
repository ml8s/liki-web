package referral

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminApplicationDecisionRequiresReason(t *testing.T) {
	store := newAdminStore(t)
	handler := NewAdminHandler(store)
	request := httptest.NewRequest(http.MethodPost, "/api/admin/referral/applications/app/decision", strings.NewReader(`{"approve":true}`))
	request.SetPathValue("id", "app")
	recorder := httptest.NewRecorder()
	handler.HandleApplicationDecision()(recorder, request)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestAdminPartnerStatusRequiresReason(t *testing.T) {
	store := newAdminStore(t)
	handler := NewAdminHandler(store)
	request := httptest.NewRequest(http.MethodPost, "/api/admin/referral/partners/kol/status", strings.NewReader(`{"status":"frozen"}`))
	request.SetPathValue("userId", "kol")
	recorder := httptest.NewRecorder()
	handler.HandlePartnerStatus()(recorder, request)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestAdminPartnerStatusFreezeAndUnfreeze(t *testing.T) {
	ctx := context.Background()
	store := newAdminStore(t)
	kol, err := store.RegisterReferrer(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewAdminHandler(store)
	request := httptest.NewRequest(http.MethodPost, "/api/admin/referral/partners/kol/status", strings.NewReader(`{"status":"frozen","reason":"manual review"}`))
	request.SetPathValue("userId", kol.UserID)
	recorder := httptest.NewRecorder()
	handler.HandlePartnerStatus()(recorder, request)
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
}
