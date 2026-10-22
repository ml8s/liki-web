package runcontrol

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/product"
)

type fakeSessions struct {
	user    auth.User
	session auth.AuthSession
	err     error
	calls   int
}

func (f *fakeSessions) GetSessionUser(context.Context, string) (auth.User, auth.AuthSession, error) {
	f.calls++
	if f.err != nil {
		return auth.User{}, auth.AuthSession{}, f.err
	}
	return f.user, f.session, nil
}

type fakeEntitlements struct {
	allowed bool
	err     error
	skills  []product.Skill
}

func (f *fakeEntitlements) HasSkill(_ context.Context, _ string, skill product.Skill) (bool, error) {
	f.skills = append(f.skills, skill)
	if f.err != nil {
		return false, f.err
	}
	return f.allowed, nil
}

func newIntrospectHandler(t *testing.T, sessions *fakeSessions, entitlements *fakeEntitlements) http.HandlerFunc {
	t.Helper()
	return NewIntrospectHandler(IntrospectConfig{
		CookieName:   "liki_session",
		Sessions:     sessions,
		Entitlements: entitlements,
	})
}

func introspectRequest(uri, cookie string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/api/internal/introspect", nil)
	if uri != "" {
		request.Header.Set("X-Forwarded-Uri", uri)
	}
	if cookie != "" {
		request.AddCookie(&http.Cookie{Name: "liki_session", Value: cookie})
	}
	return request
}

func TestIntrospectReturnsRFC7662Shape(t *testing.T) {
	expires := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	sessions := &fakeSessions{
		user:    auth.User{ID: "user_1"},
		session: auth.AuthSession{ExpiresAt: expires},
	}
	entitlements := &fakeEntitlements{allowed: true}
	handler := newIntrospectHandler(t, sessions, entitlements)

	recorder := httptest.NewRecorder()
	handler(recorder, introspectRequest("/rt/natal/ag-ui", "token-1"))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		t.Fatalf("content type = %q, want application/json", contentType)
	}
	// Caddy forward_auth copy_headers 依赖响应头注入受信身份。
	if remoteUser := recorder.Header().Get("Remote-User"); remoteUser != "user_1" {
		t.Fatalf("Remote-User header = %q, want user_1", remoteUser)
	}
	var body struct {
		Active bool   `json:"active"`
		Sub    string `json:"sub"`
		Exp    int64  `json:"exp"`
		Scope  string `json:"scope"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if !body.Active || body.Sub != "user_1" || body.Exp != expires.Unix() || body.Scope != "natal" {
		t.Fatalf("introspection body = %+v", body)
	}
	// 权益检查只针对咨询技能（专家是路由维度，不是权益维度）。
	if len(entitlements.skills) != 1 || entitlements.skills[0] != product.SkillConsultation {
		t.Fatalf("entitlement checks = %v, want [%s]", entitlements.skills, product.SkillConsultation)
	}
}

func TestIntrospectAcceptsQueryStringOnForwardedURI(t *testing.T) {
	sessions := &fakeSessions{user: auth.User{ID: "user_1"}, session: auth.AuthSession{ExpiresAt: time.Now().Add(time.Hour)}}
	handler := newIntrospectHandler(t, sessions, &fakeEntitlements{allowed: true})

	recorder := httptest.NewRecorder()
	handler(recorder, introspectRequest("/rt/fengshui/ag-ui?debug=1", "token-1"))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
}

func TestIntrospectRejectsWithoutValidSession(t *testing.T) {
	cases := map[string]struct {
		cookie   string
		sessions *fakeSessions
	}{
		"missing cookie":  {"", &fakeSessions{}},
		"unknown session": {"stale-token", &fakeSessions{err: errors.New("session not found")}},
	}
	for name, tc := range cases {
		entitlements := &fakeEntitlements{allowed: true}
		handler := newIntrospectHandler(t, tc.sessions, entitlements)

		recorder := httptest.NewRecorder()
		handler(recorder, introspectRequest("/rt/natal/ag-ui", tc.cookie))

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("%s: status = %d, want 401", name, recorder.Code)
		}
		if len(entitlements.skills) != 0 {
			t.Fatalf("%s: entitlement must not be consulted for an invalid session", name)
		}
	}
}

func TestIntrospectRejectsWithoutEntitlement(t *testing.T) {
	valid := func() *fakeSessions {
		return &fakeSessions{user: auth.User{ID: "user_1"}, session: auth.AuthSession{ExpiresAt: time.Now().Add(time.Hour)}}
	}
	cases := map[string]*fakeEntitlements{
		"expired entitlement": {allowed: false},
		"entitlement error":   {err: errors.New("db unavailable")},
	}
	for name, entitlements := range cases {
		handler := newIntrospectHandler(t, valid(), entitlements)

		recorder := httptest.NewRecorder()
		handler(recorder, introspectRequest("/rt/natal/ag-ui", "token-1"))

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("%s: status = %d, want 401", name, recorder.Code)
		}
	}
}

func TestIntrospectRejectsUnknownOrMalformedProfile(t *testing.T) {
	for _, uri := range []string{
		"",
		"/natal/ag-ui",
		"/rt/unknown/ag-ui",
		"/rt/natal/other",
		"/rt//ag-ui",
	} {
		sessions := &fakeSessions{user: auth.User{ID: "user_1"}, session: auth.AuthSession{ExpiresAt: time.Now().Add(time.Hour)}}
		handler := newIntrospectHandler(t, sessions, &fakeEntitlements{allowed: true})

		recorder := httptest.NewRecorder()
		handler(recorder, introspectRequest(uri, "token-1"))

		if recorder.Code != http.StatusForbidden {
			t.Fatalf("uri %q: status = %d, want 403", uri, recorder.Code)
		}
		if sessions.calls != 0 {
			t.Fatalf("uri %q: session must not be consulted for an invalid profile", uri)
		}
	}
}
