package partner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestReferralLinkSupportsAllowlistedTarget(t *testing.T) {
	store := newHandlerStore(t)
	kol, err := store.createKOLPartner(context.Background(), "kol")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(store, func(next http.HandlerFunc) http.HandlerFunc { return next }, true)

	recorder := httptest.NewRecorder()
	handler.HandleLink(recorder, httptest.NewRequest(http.MethodGet, "/r/"+kol.Code+"?to=%2Fzh-Hant%2Fnaming", nil))
	if recorder.Code != http.StatusFound {
		t.Fatalf("status=%d", recorder.Code)
	}
	if location := recorder.Header().Get("Location"); location != "/zh-Hant/naming" {
		t.Fatalf("location=%q", location)
	}
}

func TestReferralLinkUsesMarketingLocaleDefault(t *testing.T) {
	store := newHandlerStore(t)
	kol, err := store.createKOLPartner(context.Background(), "kol")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(store, func(next http.HandlerFunc) http.HandlerFunc { return next }, true)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/r/"+kol.Code, nil)
	request.Header.Set("Accept-Language", "ja-JP,ja;q=0.9")
	handler.HandleLink(recorder, request)

	if location := recorder.Header().Get("Location"); location != "/zh-Hant/" {
		t.Fatalf("location=%q, want /zh-Hant/", location)
	}
}

func TestReferralLinkAllowsCurrentLandingPages(t *testing.T) {
	store := newHandlerStore(t)
	kol, err := store.createKOLPartner(context.Background(), "kol")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(store, func(next http.HandlerFunc) http.HandlerFunc { return next }, true)

	for _, target := range []string{"/zh-Hant/natal", "/zh-Hans/divination", "/en/fengshui", "/zh-Hant/huangli", "/zh-Hant/hepan", "/en/liunian"} {
		recorder := httptest.NewRecorder()
		handler.HandleLink(recorder, httptest.NewRequest(http.MethodGet, "/r/"+kol.Code+"?to="+url.QueryEscape(target), nil))
		if location := recorder.Header().Get("Location"); location != target {
			t.Fatalf("target %q: location=%q", target, location)
		}
	}
}

func TestReferralLinkRejectsRetiredPartnerPages(t *testing.T) {
	store := newHandlerStore(t)
	kol, err := store.createKOLPartner(context.Background(), "kol")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(store, func(next http.HandlerFunc) http.HandlerFunc { return next }, true)

	for _, target := range []string{"/zh-Hant/partners", "/zh-Hant/mingshu"} {
		recorder := httptest.NewRecorder()
		handler.HandleLink(recorder, httptest.NewRequest(http.MethodGet, "/r/"+kol.Code+"?to="+url.QueryEscape(target), nil))
		if location := recorder.Header().Get("Location"); location == target {
			t.Fatalf("retired page %q must not be a referral target (location=%q)", target, location)
		}
	}
}

func TestReferralLinkRejectsPrivateAndExternalTargets(t *testing.T) {
	store := newHandlerStore(t)
	kol, err := store.createKOLPartner(context.Background(), "kol")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(store, func(next http.HandlerFunc) http.HandlerFunc { return next }, true)

	for _, target := range []string{
		"/app/account",
		"//evil.example",
		"https://evil.example",
	} {
		request := httptest.NewRequest(http.MethodGet, "/r/"+kol.Code+"?to="+target, nil)
		recorder := httptest.NewRecorder()
		handler.HandleLink(recorder, request)
		location := recorder.Header().Get("Location")
		if strings.Contains(location, "evil") || strings.HasPrefix(location, "/app/account") {
			t.Fatalf("target %q produced unsafe location %q", target, location)
		}
	}
}
