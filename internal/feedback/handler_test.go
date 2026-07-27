package feedback

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func rateLimit(next http.HandlerFunc) http.HandlerFunc { return next }

func TestHandleCreate_Success(t *testing.T) {
	s := setup(t)

	body := `{"category":"workflow","message":"步骤4不清晰","context":"{\"step\":4}"}`
	req := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleCreate(s, rateLimit)(w, req)

	if w.Code != 201 {
		t.Fatalf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	var res map[string]string
	json.NewDecoder(w.Body).Decode(&res)
	if res["id"] == "" || res["status"] != "pending" {
		t.Fatalf("bad response: %+v", res)
	}
}

func TestHandleCreate_MissingMessage(t *testing.T) {
	s := setup(t)

	body := `{"category":"workflow"}`
	req := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleCreate(s, rateLimit)(w, req)

	if w.Code != 400 {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreate_InvalidJSON(t *testing.T) {
	s := setup(t)

	req := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleCreate(s, rateLimit)(w, req)

	if w.Code != 400 {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestHandleCreate_DefaultCategory(t *testing.T) {
	s := setup(t)

	body := `{"message":"test only"}`
	req := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleCreate(s, rateLimit)(w, req)

	if w.Code != 201 {
		t.Fatalf("want 201, got %d", w.Code)
	}
	var res map[string]string
	json.NewDecoder(w.Body).Decode(&res)

	list, _ := s.List("", 10)
	if len(list) != 1 || list[0].Category != "other" {
		t.Fatalf("default category failed: %+v", list)
	}
}

func TestHandleList_RequiresAuth(t *testing.T) {
	s := setup(t)

	req := httptest.NewRequest("GET", "/api/feedback", nil)
	w := httptest.NewRecorder()

	HandleList(s, "secret-key")(w, req)

	if w.Code != 403 {
		t.Fatalf("want 403, got %d", w.Code)
	}
}

func TestHandleList_WithAuth(t *testing.T) {
	s := setup(t)
	s.Create(&Feedback{Message: "test"}) //nolint:errcheck

	req := httptest.NewRequest("GET", "/api/feedback?status=pending", nil)
	req.Header.Set("Authorization", "Bearer my-key")
	w := httptest.NewRecorder()

	HandleList(s, "my-key")(w, req)

	if w.Code != 200 {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var list []Feedback
	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 1 || list[0].Message != "test" {
		t.Fatalf("bad list: %+v", list)
	}
}

func TestHandleList_EmptyAuthKey(t *testing.T) {
	s := setup(t)

	req := httptest.NewRequest("GET", "/api/feedback", nil)
	w := httptest.NewRecorder()

	HandleList(s, "")(w, req)

	if w.Code != 403 {
		t.Fatalf("want 403, got %d", w.Code)
	}
}

func TestHandleUpdate_Success(t *testing.T) {
	s := setup(t)
	s.Create(&Feedback{Message: "fix me"}) //nolint:errcheck
	list, _ := s.List("", 10)
	id := list[0].ID

	body, _ := json.Marshal(UpdateRequest{Status: "done", IssueURL: "https://github.com/ml8s/liki/issues/1"})
	req := httptest.NewRequest("PATCH", "/api/feedback/"+id, bytes.NewReader(body))
	req.Header.Set("Authorization", "my-key")
	req.SetPathValue("id", id)
	w := httptest.NewRecorder()

	HandleUpdate(s, "my-key")(w, req)

	if w.Code != 200 {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleUpdate_NotFound(t *testing.T) {
	s := setup(t)

	body, _ := json.Marshal(UpdateRequest{Status: "done"})
	req := httptest.NewRequest("PATCH", "/api/feedback/no-such", bytes.NewReader(body))
	req.Header.Set("Authorization", "my-key")
	req.SetPathValue("id", "no-such")
	w := httptest.NewRecorder()

	HandleUpdate(s, "my-key")(w, req)

	if w.Code != 404 {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestHandleUpdate_NoAuth(t *testing.T) {
	s := setup(t)
	body, _ := json.Marshal(UpdateRequest{Status: "done"})
	req := httptest.NewRequest("PATCH", "/api/feedback/x", bytes.NewReader(body))
	w := httptest.NewRecorder()

	HandleUpdate(s, "my-key")(w, req)

	if w.Code != 403 {
		t.Fatalf("want 403, got %d", w.Code)
	}
}

func setup(t *testing.T) *Store {
	t.Helper()
	db := openTestDB(t)
	s, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
