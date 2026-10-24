package birthprofile

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ml8s/liki-web/internal/auth"
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
	wrap(h).ServeHTTP(rr, req)
	return rr
}

func wrap(h *Handler) http.Handler {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux, func(next http.HandlerFunc) http.HandlerFunc {
		return next
	})
	return mux
}

func completePayload(nickname string) map[string]any {
	return map[string]any{
		"nickname":            nickname,
		"third_party_consent": true,
		"birth": map[string]any{
			"gender": "female", "date": "1958-05-12", "time": "06:00",
			"location": map[string]any{
				"place": "上海", "place_id": "CN-310000", "longitude": 121.4737, "timezone": "Asia/Shanghai",
			},
		},
	}
}

func TestHTTPCreateAndGet(t *testing.T) {
	h := newHandlerFixture(t)
	rr := doJSON(t, h, http.MethodPost, "/api/birth-profiles", completePayload("妈妈"))
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String())
	}
	var env struct {
		Data BirthProfile `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Data.ID == "" || env.Data.Birth.Instant == nil || env.Data.Birth.Date != "1958-05-12" {
		t.Fatalf("unexpected profile: %+v", env.Data)
	}
	if env.Data.Birth.Location.PlaceID != "CN-310000" {
		t.Fatalf("place id = %q", env.Data.Birth.Location.PlaceID)
	}
	if env.Data.Birth.Location.LocationVersion != 2 {
		t.Fatalf("location version = %d", env.Data.Birth.Location.LocationVersion)
	}

	rr = doJSON(t, h, http.MethodGet, "/api/birth-profiles/"+env.Data.ID, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHTTPValidation(t *testing.T) {
	h := newHandlerFixture(t)
	rr := doJSON(t, h, http.MethodPost, "/api/birth-profiles", map[string]any{
		"nickname": "",
		"birth": map[string]any{
			"gender": "female", "date": "1958-05-12", "time": "06:00",
			"location": map[string]any{
				"place": "上海", "place_id": "CN-310000", "longitude": 121.4737, "timezone": "Asia/Shanghai",
			},
		},
	})
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHTTPRejectsRetiredSemanticFields(t *testing.T) {
	h := newHandlerFixture(t)
	payload := completePayload("妈妈")
	payload["calendar"] = "solar"
	payload["birth"].(map[string]any)["birth_time_known"] = "known"

	rr := doJSON(t, h, http.MethodPost, "/api/birth-profiles", payload)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for retired semantic fields, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHTTPUpdateRequiresExpectedVersion(t *testing.T) {
	h := newHandlerFixture(t)
	rr := doJSON(t, h, http.MethodPost, "/api/birth-profiles", completePayload("妈妈"))
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String())
	}
	var created struct {
		Data BirthProfile `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	payload := completePayload("妈妈（新）")
	rr = doJSON(t, h, http.MethodPatch, "/api/birth-profiles/"+created.Data.ID, payload)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 without expected_version, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHTTPRequiresCompleteChartInput(t *testing.T) {
	h := newHandlerFixture(t)
	payload := completePayload("妈妈")
	payload["birth"].(map[string]any)["location"].(map[string]any)["timezone"] = nil
	rr := doJSON(t, h, http.MethodPost, "/api/birth-profiles", payload)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for incomplete input, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHTTPUpdateConflict(t *testing.T) {
	h := newHandlerFixture(t)
	rr := doJSON(t, h, http.MethodPost, "/api/birth-profiles", completePayload("妈妈"))
	var env struct {
		Data BirthProfile `json:"data"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &env)

	payload := completePayload("改名")
	payload["expected_version"] = 999
	rr = doJSON(t, h, http.MethodPatch, "/api/birth-profiles/"+env.Data.ID, payload)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHTTPListAndDelete(t *testing.T) {
	h := newHandlerFixture(t)
	for _, nickname := range []string{"甲", "乙", "丙"} {
		payload := completePayload(nickname)
		payload["birth"].(map[string]any)["gender"] = "male"
		payload["birth"].(map[string]any)["date"] = "1990-01-01"
		payload["birth"].(map[string]any)["time"] = "12:00"
		doJSON(t, h, http.MethodPost, "/api/birth-profiles", payload)
	}
	rr := doJSON(t, h, http.MethodGet, "/api/birth-profiles", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("list status=%d", rr.Code)
	}
	var list struct {
		Data struct {
			Profiles []BirthProfile `json:"profiles"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Data.Profiles) != 3 {
		t.Fatalf("expected 3 profiles, got %d", len(list.Data.Profiles))
	}
	id := list.Data.Profiles[0].ID

	rr = doJSON(t, h, http.MethodDelete, "/api/birth-profiles/"+id, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete status=%d", rr.Code)
	}
	rr = doJSON(t, h, http.MethodGet, "/api/birth-profiles/"+id, nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", rr.Code)
	}
}
