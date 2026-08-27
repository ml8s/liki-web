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

	list, _ := s.List("", 10) //nolint:errcheck
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
	s.Create(&Feedback{Message: "test"}) //nolint:errcheck //nolint:errcheck

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

// 空结果应序列化为 [] 而非 null（客户端不需判空）
func TestHandleList_EmptyReturnsEmptyArray(t *testing.T) {
	s := setup(t)

	req := httptest.NewRequest("GET", "/api/feedback", nil)
	req.Header.Set("Authorization", "Bearer my-key")
	w := httptest.NewRecorder()

	HandleList(s, "my-key")(w, req)

	if w.Code != 200 {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if body := strings.TrimSpace(w.Body.String()); body != "[]" {
		t.Fatalf("want [], got %s", body)
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
	s.Create(&Feedback{Message: "fix me"}) //nolint:errcheck //nolint:errcheck
	list, _ := s.List("", 10)              //nolint:errcheck
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

// 回归保护：GBK 等非 UTF-8 body 曾被 encoding/json 静默替换为 U+FFFD 入库，
// 两条用户反馈（3376740b/8d453a6a）因此不可逆损坏。现在必须 400 拒绝且不落库。
func TestHandleCreate_RejectsNonUTF8(t *testing.T) {
	s := setup(t)

	// 「用户反馈」四字的 GBK 编码（非法 UTF-8）
	gbkBody := append([]byte(`{"category":"bug","message":"`), 0xD3, 0xC3, 0xBB, 0xA7, 0xB7, 0xB4, 0xC0, 0xA1)
	gbkBody = append(gbkBody, []byte(`"}`)...)
	req := httptest.NewRequest("POST", "/api/feedback", bytes.NewReader(gbkBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleCreate(s, rateLimit)(w, req)

	if w.Code != 400 {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "UTF-8") {
		t.Fatalf("错误信息应指明 UTF-8，got %q", w.Body.String())
	}
	list, err := s.List("", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("非 UTF-8 反馈不应入库，got %d 条", len(list))
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
