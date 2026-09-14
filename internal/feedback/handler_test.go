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

const validFeedbackV1 = `{"schema_version":"feedback-v1","meta":{"source":"skill-agent","skill":"liki-bazi","skill_version":"2026.09.14.0","engine_version":"2026.09.14.0"},"problem":{"type":"clarity","severity":"medium","summary":"resolved_period 解释入口不直观"}}`

func TestHandleCreate_Success(t *testing.T) {
	s := setup(t)

	req := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(validFeedbackV1))
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
	list, _ := s.List("", 10) //nolint:errcheck
	if len(list) != 1 || list[0].Payload != validFeedbackV1 {
		t.Fatalf("payload not stored verbatim: %+v", list)
	}
}

func TestHandleCreate_Deduplicates(t *testing.T) {
	s := setup(t)
	post := func() string {
		req := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(validFeedbackV1))
		w := httptest.NewRecorder()
		HandleCreate(s, rateLimit)(w, req)
		if w.Code != 201 {
			t.Fatalf("want 201, got %d", w.Code)
		}
		var res map[string]string
		json.NewDecoder(w.Body).Decode(&res)
		return res["id"]
	}
	first := post()
	second := post()
	if first == "" || first != second {
		t.Fatalf("identical feedback must dedup to same id: %q vs %q", first, second)
	}
	list, _ := s.List("", 10) //nolint:errcheck
	if len(list) != 1 {
		t.Fatalf("dedup failed: %d rows", len(list))
	}
}

func TestHandleCreate_RejectsInvalidJSON(t *testing.T) {
	s := setup(t)
	req := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(`{invalid`))
	w := httptest.NewRecorder()
	HandleCreate(s, rateLimit)(w, req)
	if w.Code != 400 {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestHandleCreate_RejectsNonObject(t *testing.T) {
	s := setup(t)
	// JSON 数组不是对象——极简校验拒绝。
	req := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(`[1,2,3]`))
	w := httptest.NewRecorder()
	HandleCreate(s, rateLimit)(w, req)
	if w.Code != 400 {
		t.Fatalf("want 400 for non-object, got %d", w.Code)
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
	s.Create(&Feedback{Payload: validFeedbackV1}) //nolint:errcheck

	req := httptest.NewRequest("GET", "/api/feedback?status=pending", nil)
	req.Header.Set("Authorization", "Bearer my-key")
	w := httptest.NewRecorder()
	HandleList(s, "my-key")(w, req)

	if w.Code != 200 {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var list []Feedback
	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 1 || list[0].Payload != validFeedbackV1 {
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
	s.Create(&Feedback{Payload: validFeedbackV1}) //nolint:errcheck
	list, _ := s.List("", 10)                     //nolint:errcheck
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

// 回归保护：非 UTF-8 body 必须 400 且不落库。
func TestHandleCreate_RejectsNonUTF8(t *testing.T) {
	s := setup(t)
	// 「问题」两字的 GBK 编码（非法 UTF-8）放进 problem.summary
	gbkBody := append([]byte(`{"problem":{"summary":"`), 0xC, 0xCC, 0xCC, 0xE2)
	gbkBody = append(gbkBody, []byte(`"}}`)...)
	req := httptest.NewRequest("POST", "/api/feedback", bytes.NewReader(gbkBody))
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
