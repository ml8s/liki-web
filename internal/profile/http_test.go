package profile

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"liki-web/internal/auth"
)

func newHandlerFixture(t *testing.T) *Handler {
	t.Helper()
	store := NewStore(openTestDB(t))
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return NewHandler(store, func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user := auth.User{ID: "user_1", Email: "u@example.com", Status: auth.UserActive}
			r = r.WithContext(auth.WithAuth(r.Context(), user, auth.AuthSession{ID: "session"}, "token"))
			next(w, r)
		}
	})
}

func doJSON(t *testing.T, h *Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	handler := wrap(h)
	handler.ServeHTTP(rr, req)
	return rr
}

// wrap exposes the profile routes on a mux, mirroring RegisterRoutes with a
// pass-through limiter.
func wrap(h *Handler) http.Handler {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux, func(next http.HandlerFunc) http.HandlerFunc { return next })
	return mux
}

func TestHTTPCreateAndGet(t *testing.T) {
	h := newHandlerFixture(t)
	rr := doJSON(t, h, http.MethodPost, "/api/profiles", map[string]any{
		"nickname": "妈妈", "gender": "female", "calendar": "lunar",
		"birth_date": "1958-闰5-12", "birth_time": "06:00", "birth_time_known": "known",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String())
	}
	var env struct {
		Data Profile `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	id := env.Data.ID
	if id == "" {
		t.Fatal("missing profile id")
	}

	rr = doJSON(t, h, http.MethodGet, "/api/profiles/"+id, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHTTPValidationError(t *testing.T) {
	h := newHandlerFixture(t)
	rr := doJSON(t, h, http.MethodPost, "/api/profiles", map[string]any{
		"nickname": "", "gender": "female", "calendar": "lunar",
		"birth_date": "1958-闰5-12", "birth_time": "06:00", "birth_time_known": "known",
	})
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHTTPUpdateConflict(t *testing.T) {
	h := newHandlerFixture(t)
	rr := doJSON(t, h, http.MethodPost, "/api/profiles", map[string]any{
		"nickname": "妈妈", "gender": "female", "calendar": "lunar",
		"birth_date": "1958-闰5-12", "birth_time": "06:00", "birth_time_known": "known",
	})
	var env struct {
		Data Profile `json:"data"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &env)

	rr = doJSON(t, h, http.MethodPatch, "/api/profiles/"+env.Data.ID, map[string]any{
		"expected_version": 999, "nickname": "改名", "gender": "female", "calendar": "lunar",
		"birth_date": "1958-闰5-12", "birth_time": "06:00", "birth_time_known": "known",
	})
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHTTPListAndDelete(t *testing.T) {
	h := newHandlerFixture(t)
	for _, nickname := range []string{"甲", "乙", "丙"} {
		doJSON(t, h, http.MethodPost, "/api/profiles", map[string]any{
			"nickname": nickname, "gender": "male", "calendar": "solar",
			"birth_date": "1990-01-01", "birth_time": "12:00", "birth_time_known": "known",
		})
	}
	rr := doJSON(t, h, http.MethodGet, "/api/profiles", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("list status=%d", rr.Code)
	}
	var list struct {
		Data struct {
			Profiles []Profile `json:"profiles"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Data.Profiles) != 3 {
		t.Fatalf("expected 3 profiles, got %d", len(list.Data.Profiles))
	}
	id := list.Data.Profiles[0].ID

	rr = doJSON(t, h, http.MethodDelete, "/api/profiles/"+id, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete status=%d", rr.Code)
	}
	rr = doJSON(t, h, http.MethodGet, "/api/profiles/"+id, nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", rr.Code)
	}
}
