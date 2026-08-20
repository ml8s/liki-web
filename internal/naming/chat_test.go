package naming

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"liki-web/internal/agent"
	"liki-web/internal/httputil"
	"liki-web/internal/llm"

	"github.com/google/uuid"
)

func chatSeedPaid(t *testing.T, s *Store) string {
	t.Helper()
	oid := uuid.New().String()
	if err := s.CreateOrder(context.Background(), oid, "naming", 990, "CNY", "test@t.com", "zh-Hans"); err != nil {
		t.Fatalf("create order: %v", err)
	}
	s.MarkPaidIdempotent(context.Background(), oid, "pay-1") //nolint:errcheck
	s.OverrideChatExpiresAt(context.Background(), oid, "2099-12-31") //nolint:errcheck
	// Seed 2 chat messages so history is non-empty
	s.CreateChatMessage(context.Background(), oid, "user", "你好")        //nolint:errcheck
	s.CreateChatMessage(context.Background(), oid, "assistant", "你好！请告诉我你的出生日期") //nolint:errcheck
	return oid
}

func chatJWT(oid string) *http.Cookie {
	w := httptest.NewRecorder()
	httputil.SetJWTCookie(w, "test@t.com", oid) //nolint:errcheck
	return w.Result().Cookies()[0]
}

func chatHandler(store *Store, mockResps []*llm.ChatResult) (http.HandlerFunc, *agent.MockLLM) {
	mockLLM := &agent.MockLLM{ToolResps: mockResps}
	mockTools := &agent.MockToolRegistry{}
	chat := agent.NewAgent(mockLLM, mockTools, "test {locale}\n{phase_instruction}")
	return HandleChat(chat, store), mockLLM
}

// TestHandleChat_Success verifies full chat flow: SSE events + DB persistence.
func TestHandleChat_Success(t *testing.T) {
	
	t.Setenv("JWT_SECRET", "test-secret")
	s := newTestStore(t)
	oid := chatSeedPaid(t, s)

	handler, _ := chatHandler(s, []*llm.ChatResult{
		agent.ChatRes(nil, "回复内容"),
	})

	body := `{"message":"测试消息","lang":"zh-Hans"}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	r.AddCookie(chatJWT(oid))
	w := httptest.NewRecorder()
	handler(w, r)

	// Check the order was debited (message saved)
	msgs, err := s.LoadChatHistory(context.Background(), oid)
	if err != nil {
		t.Fatalf("LoadChatHistory: %v", err)
	}
	if len(msgs) < 3 {
		t.Fatalf("expected >=3 messages (history 2 + new user + assistant), got %d", len(msgs))
	}
	if msgs[len(msgs)-1].Role != "assistant" {
		t.Errorf("last message role = %q, want assistant", msgs[len(msgs)-1].Role)
	}
	if msgs[len(msgs)-2].Role != "user" {
		t.Errorf("second-last message role = %q, want user", msgs[len(msgs)-2].Role)
	}
	if msgs[len(msgs)-2].Content != "测试消息" {
		t.Errorf("user message content = %q", msgs[len(msgs)-2].Content)
	}

	// Check SSE format
	scanner := bufio.NewScanner(w.Body)
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			var evt agent.ChatEvent
			if err := json.Unmarshal([]byte(line[6:]), &evt); err == nil && evt.Type == "text-delta" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("no text-delta SSE event found in response")
	}
}

// TestHandleChat_NoJWT returns 401 when no cookie is sent.
func TestHandleChat_NoJWT(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	s := newTestStore(t)
	handler, _ := chatHandler(s, nil)
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"message":"hi"}`))
	w := httptest.NewRecorder()
	handler(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// TestHandleChat_InvalidJWT returns 401 for a tampered token.
func TestHandleChat_InvalidJWT(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	s := newTestStore(t)
	handler, _ := chatHandler(s, nil)
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"message":"hi"}`))
	r.AddCookie(&http.Cookie{Name: "liki_token", Value: "tampered"})
	w := httptest.NewRecorder()
	handler(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// TestHandleChat_OrderNotPaid returns 403 when order is pending.
func TestHandleChat_OrderNotPaid(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	s := newTestStore(t)
	oid := uuid.New().String()
	s.CreateOrder(context.Background(), oid, "naming", 990, "CNY", "test@t.com", "zh-Hans") //nolint:errcheck
	handler, _ := chatHandler(s, nil)
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"message":"hi"}`))
	r.AddCookie(chatJWT(oid))
	w := httptest.NewRecorder()
	handler(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

// TestHandleChat_OrderExpired returns 403 when chat_expires_at is in the past.
func TestHandleChat_OrderExpired(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	s := newTestStore(t)
	oid := uuid.New().String()
	s.CreateOrder(context.Background(), oid, "naming", 990, "CNY", "test@t.com", "zh-Hans") //nolint:errcheck
	s.MarkPaidIdempotent(context.Background(), oid, "pay-1") //nolint:errcheck
	s.OverrideChatExpiresAt(context.Background(), oid, "2020-01-01 00:00:00") //nolint:errcheck
	handler, _ := chatHandler(s, nil)
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"message":"hi"}`))
	r.AddCookie(chatJWT(oid))
	w := httptest.NewRecorder()
	handler(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

// TestHandleChat_EmptyMessage returns 400 for empty message.
func TestHandleChat_EmptyMessage(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	s := newTestStore(t)
	oid := chatSeedPaid(t, s)
	handler, _ := chatHandler(s, nil)
	// Send with empty message
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"message":""}`))
	r.AddCookie(chatJWT(oid))
	w := httptest.NewRecorder()
	handler(w, r)
	if w.Code != http.StatusUnprocessableEntity && w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 422 or 400, body: %s", w.Code, w.Body.String())
	}
}

// TestHandleChat_LLMError writes error SSE event, doesn't crash.
func TestHandleChat_LLMError(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	s := newTestStore(t)
	oid := chatSeedPaid(t, s)

	// Mock LLM returns an error
	mockLLM := &agent.MockLLM{ToolResps: []*llm.ChatResult{agent.ChatRes(nil, "")}, ToolErrs: []error{fmt.Errorf("llm down")}}
	mockTools := &agent.MockToolRegistry{}
	chat := agent.NewAgent(mockLLM, mockTools, "test {locale}\n{phase_instruction}")
	handler := HandleChat(chat, s)

	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"message":"hi"}`))
	r.AddCookie(chatJWT(oid))
	w := httptest.NewRecorder()
	handler(w, r)

	// Should have written an error SSE event
	scanner := bufio.NewScanner(w.Body)
	found := false
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), `"type":"error"`) {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error SSE event")
	}
}

// TestHandleChat_ExpiresSoon is within expiry window — should work.
func TestHandleChat_ExpiresSoon(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	s := newTestStore(t)
	oid := uuid.New().String()
	s.CreateOrder(context.Background(), oid, "naming", 990, "CNY", "test@t.com", "zh-Hans") //nolint:errcheck
	s.MarkPaidIdempotent(context.Background(), oid, "pay-1") //nolint:errcheck
	// Set expiry to 1 hour from now
	future := time.Now().Add(1 * time.Hour).Format("2006-01-02 15:04:05")
	s.OverrideChatExpiresAt(context.Background(), oid, future) //nolint:errcheck

	handler, _ := chatHandler(s, []*llm.ChatResult{
		agent.ChatRes(nil, "ok"),
	})
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"message":"hi"}`))
	r.AddCookie(chatJWT(oid))
	w := httptest.NewRecorder()
	handler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
}

// TestHandleChat_EnglishLocale sets lang to en.
func TestHandleChat_EnglishLocale(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	s := newTestStore(t)
	oid := chatSeedPaid(t, s)

	handler, _ := chatHandler(s, []*llm.ChatResult{
		agent.ChatRes(nil, "hello"),
	})
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"message":"hello","lang":"en"}`))
	r.AddCookie(chatJWT(oid))
	w := httptest.NewRecorder()
	handler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}


// ── HandleChatHistory ──

func TestHandleChatHistory_Success(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	s := newTestStore(t)
	oid := chatSeedPaid(t, s)

	handler := HandleChatHistory(s)
	r := httptest.NewRequest("GET", "/api/naming/orders/"+oid+"/messages", nil)
	r.SetPathValue("id", oid)
	r.AddCookie(chatJWT(oid))
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp struct {
		Data []ChatMessage `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Data) < 2 {
		t.Errorf("expected >=2 messages, got %d", len(resp.Data))
	}
}

func TestHandleChatHistory_NoJWT(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	s := newTestStore(t)
	handler := HandleChatHistory(s)
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// ── HandleDeleteChatHistory ──

func TestHandleDeleteChatHistory_Success(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	s := newTestStore(t)
	oid := chatSeedPaid(t, s)

	handler := HandleDeleteChatHistory(s)
	r := httptest.NewRequest("DELETE", "/api/naming/orders/"+oid+"/messages", nil)
	r.SetPathValue("id", oid)
	r.AddCookie(chatJWT(oid))
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	// Verify messages are deleted from DB
	msgs, _ := s.LoadChatHistory(context.Background(), oid) //nolint:errcheck
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages after delete, got %d", len(msgs))
	}
}

func TestHandleDeleteChatHistory_NoJWT(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	s := newTestStore(t)
	handler := HandleDeleteChatHistory(s)
	r := httptest.NewRequest("DELETE", "/", nil)
	w := httptest.NewRecorder()
	handler(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// TestHandleChat_GreetingPersisted verifies greeting context is persisted as
// an assistant message AND injected into LLM context on first turn (empty history).
func TestHandleChat_GreetingPersisted(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	s := newTestStore(t)
	oid := uuid.New().String()
	if err := s.CreateOrder(context.Background(), oid, "naming", 990, "CNY", "test@t.com", "zh-Hans"); err != nil {
		t.Fatalf("create order: %v", err)
	}
	s.MarkPaidIdempotent(context.Background(), oid, "pay-g")          //nolint:errcheck
	s.OverrideChatExpiresAt(context.Background(), oid, "2099-12-31") //nolint:errcheck

	greeting := "你好，我是 Liki 灵机。请问这次需要哪类服务？① 宝宝起名 ② 改名 ③ 评估"
	handler, mockLLM := chatHandler(s, []*llm.ChatResult{agent.ChatRes(nil, "回复")})
	body := `{"message":"1","lang":"zh-Hans","context":"` + greeting + `"}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	r.AddCookie(chatJWT(oid))
	w := httptest.NewRecorder()
	handler(w, r)

	// 1. greeting must be persisted as assistant message (before user msg).
	hist, err := s.LoadChatHistory(context.Background(), oid)
	if err != nil {
		t.Fatalf("LoadChatHistory: %v", err)
	}
	if len(hist) < 3 {
		t.Fatalf("expected >=3 messages (greeting + user + assistant reply), got %d", len(hist))
	}
	if hist[0].Role != "assistant" || hist[0].Content != greeting {
		t.Errorf("hist[0] = %s/%q, want assistant greeting", hist[0].Role, hist[0].Content)
	}
	if hist[1].Role != "user" || hist[1].Content != "1" {
		t.Errorf("hist[1] = %s/%q, want user 1", hist[1].Role, hist[1].Content)
	}

	// 2. LLM context must contain greeting as assistant message.
	if len(mockLLM.Calls) == 0 {
		t.Fatal("MockLLM was never called")
	}
	msgs := mockLLM.Calls[0]
	var found bool
	for _, m := range msgs {
		if m.Role == llm.RoleAssistant && m.Content == greeting {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("greeting not injected into LLM context; got: %+v", msgs)
	}
}

// TestHandleChat_GreetingNotPersisted_WhenHistory verifies greeting is NOT
// persisted/injected again when history already exists (subsequent turns).
func TestHandleChat_GreetingNotPersisted_WhenHistory(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	s := newTestStore(t)
	oid := chatSeedPaid(t, s) // seeds 2 messages → history non-empty

	handler, mockLLM := chatHandler(s, []*llm.ChatResult{agent.ChatRes(nil, "回复")})
	body := `{"message":"2","lang":"zh-Hans","context":"greeting-text"}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	r.AddCookie(chatJWT(oid))
	w := httptest.NewRecorder()
	handler(w, r)

	// history: 2 seeded + 1 new user + 1 assistant reply = 4; greeting NOT added.
	hist, err := s.LoadChatHistory(context.Background(), oid)
	if err != nil {
		t.Fatalf("LoadChatHistory: %v", err)
	}
	if len(hist) != 4 {
		t.Fatalf("expected 4 messages (2 history + user + assistant), got %d", len(hist))
	}
	for _, m := range hist {
		if m.Content == "greeting-text" {
			t.Errorf("greeting must NOT be persisted when history non-empty")
		}
	}

	// LLM context must NOT contain greeting-text.
	if len(mockLLM.Calls) == 0 {
		t.Fatal("MockLLM was never called")
	}
	for _, m := range mockLLM.Calls[0] {
		if m.Content == "greeting-text" {
			t.Errorf("greeting must NOT be injected when history non-empty")
		}
	}
}
