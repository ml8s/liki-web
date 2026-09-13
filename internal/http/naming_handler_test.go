package http

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"liki-web/internal/agent"
	"liki-web/internal/httputil"
	"liki-web/internal/llm"
	"liki-web/internal/naming"
	"liki-web/internal/product"
)

func setupNamingHandler(t *testing.T, store *naming.Store) http.HandlerFunc {
	t.Helper()
	mockLLM := &agent.MockLLM{
		ToolResps: []*llm.ChatResult{agent.ChatRes(nil, "你好，请告诉我你的出生日期")},
	}
	mockTools := &agent.MockToolRegistry{}
	chat := agent.NewAgent(mockLLM, mockTools, "test {locale}\n{phase_instruction}")
	return naming.HandleChat(chat, store)
}

func TestNamingHandler_NoJWT(t *testing.T) {
	store := newAuthTestStore(t)
	handler := setupNamingHandler(t, store)

	r := httptest.NewRequest("POST", "/api/naming/agent/chat", strings.NewReader(`{"message":"你好"}`))
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestNamingHandler_OrderNotPaid(t *testing.T) {
	store := newAuthTestStore(t)
	if err := store.CreateOrder(context.Background(), "order-1", "naming", 990, "CNY", "", "zh-Hans"); err != nil {
		t.Fatalf("create order: %v", err)
	}
	handler := setupNamingHandler(t, store)

	r := httptest.NewRequest("POST", "/api/naming/agent/chat", strings.NewReader(`{"message":"你好"}`))
	setJWTForTest(t, r, "user@example.com", "order-1")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestNamingHandler_Expired(t *testing.T) {
	store := newAuthTestStore(t)
	seedActiveOrder(t, store, "order-1", "user@example.com", product.ProductNaming, "2020-01-01 00:00:00")
	handler := setupNamingHandler(t, store)

	r := httptest.NewRequest("POST", "/api/naming/agent/chat", strings.NewReader(`{"message":"你好"}`))
	setJWTForTest(t, r, "user@example.com", "order-1")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestNamingHandler_EmptyMessage(t *testing.T) {
	store := newAuthTestStore(t)
	seedActiveOrder(t, store, "order-1", "user@example.com", product.ProductNaming, "2027-01-01 00:00:00")
	handler := setupNamingHandler(t, store)

	r := httptest.NewRequest("POST", "/api/naming/agent/chat", strings.NewReader(`{"message":""}`))
	setJWTForTest(t, r, "user@example.com", "order-1")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", w.Code)
	}
}

func TestNamingHandler_SSEStream(t *testing.T) {
	store := newAuthTestStore(t)
	seedActiveOrder(t, store, "order-1", "user@example.com", product.ProductNaming, "2027-01-01 00:00:00")

	mockLLM := &agent.MockLLM{
		ToolResps: []*llm.ChatResult{agent.ChatRes(nil, "你好，请告诉我你的出生日期")},
	}
	mockTools := &agent.MockToolRegistry{}
	chat := agent.NewAgent(mockLLM, mockTools, "test {locale}\n{phase_instruction}")
	handler := naming.HandleChat(chat, store)

	r := httptest.NewRequest("POST", "/api/naming/agent/chat", strings.NewReader(`{"message":"你好"}`))
	setJWTForTest(t, r, "user@example.com", "order-1")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}

	var events []agent.ChatEvent
	scanner := bufio.NewScanner(w.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var ev agent.ChatEvent
			if err := json.Unmarshal([]byte(data), &ev); err != nil {
				continue
			}
			events = append(events, ev)
		}
	}

	hasThinking := false
	hasTextDelta := false
	var textContent string
	for _, ev := range events {
		switch ev.Type {
		case agent.EventThinking:
			hasThinking = true
		case agent.EventTextDelta:
			hasTextDelta = true
			textContent += ev.Content
		}
	}
	if !hasThinking {
		t.Error("missing thinking SSE event")
	}
	if !hasTextDelta {
		t.Error("missing text-delta SSE event")
	}
	if textContent != "你好，请告诉我你的出生日期" {
		t.Errorf("text-delta content = %q, want %q", textContent, "你好，请告诉我你的出生日期")
	}
}

func TestNamingHandler_ToolCallPhaseEvent(t *testing.T) {
	store := newAuthTestStore(t)
	seedActiveOrder(t, store, "order-1", "user@example.com", product.ProductNaming, "2027-01-01 00:00:00")

	mockLLM := &agent.MockLLM{
		ToolResps: []*llm.ChatResult{
			agent.ChatRes([]llm.ToolCall{agent.ToolCall("compute_chart", `{"birth":{"time":"1984-02-15T08:00:00+08:00","longitude":116.4},"gender":"male"}`)}, "好的，让我先推算八字"),
			agent.ChatRes(nil, "基于您的八字，我建议…"),
		},
	}
	mockTools := &agent.MockToolRegistry{}
	chat := agent.NewAgent(mockLLM, mockTools, "test {locale}\n{phase_instruction}")
	handler := naming.HandleChat(chat, store)

	r := httptest.NewRequest("POST", "/api/naming/agent/chat", strings.NewReader(`{"message":"帮我起名"}`))
	setJWTForTest(t, r, "user@example.com", "order-1")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var events []agent.ChatEvent
	scanner := bufio.NewScanner(w.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var ev agent.ChatEvent
			if err := json.Unmarshal([]byte(data), &ev); err != nil {
				continue
			}
			events = append(events, ev)
		}
	}

	hasThinking := false
	hasPhase := false
	hasTextDelta := false
	var allText string
	for _, ev := range events {
		switch ev.Type {
		case agent.EventThinking:
			hasThinking = true
		case agent.EventPhase:
			hasPhase = true
			if ev.Content != "正在计算命理数据…" {
				t.Errorf("phase content = %q, want %q", ev.Content, "正在计算命理数据…")
			}
		case agent.EventTextDelta:
			hasTextDelta = true
			allText += ev.Content
		}
	}
	if !hasThinking {
		t.Error("missing thinking event")
	}
	if !hasPhase {
		t.Error("missing phase event for compute_chart tool call")
	}
	if !hasTextDelta {
		t.Error("missing text-delta event")
	}
	if !strings.Contains(allText, "好的，让我先推算八字") {
		t.Errorf("missing first response content, got: %q", allText)
	}
	if !strings.Contains(allText, "基于您的八字，我建议…") {
		t.Errorf("missing second response content, got: %q", allText)
	}
}

func TestNamingHandler_LLMError(t *testing.T) {
	store := newAuthTestStore(t)
	seedActiveOrder(t, store, "order-1", "user@example.com", product.ProductNaming, "2027-01-01 00:00:00")

	mockLLM := &agent.MockLLM{
		ToolErrs: []error{fmt.Errorf("LLM service unavailable")},
	}
	mockTools := &agent.MockToolRegistry{}
	chat := agent.NewAgent(mockLLM, mockTools, "test {locale}\n{phase_instruction}")
	handler := naming.HandleChat(chat, store)

	r := httptest.NewRequest("POST", "/api/naming/agent/chat", strings.NewReader(`{"message":"帮我起名"}`))
	setJWTForTest(t, r, "user@example.com", "order-1")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var hasError bool
	scanner := bufio.NewScanner(w.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var ev agent.ChatEvent
			if err := json.Unmarshal([]byte(data), &ev); err != nil {
				continue
			}
			if ev.Type == agent.EventError {
				hasError = true
			}
		}
	}
	if !hasError {
		t.Error("missing error event for LLM failure")
	}

	// Verify user message was persisted even though LLM failed.
	msgs, err := store.LoadChatHistory(context.Background(), "order-1")
	if err != nil {
		t.Fatalf("LoadChatHistory: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("message count = %d, want 1 (user message persisted before error)", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "帮我起名" {
		t.Errorf("msg[0] = {%q, %q}, want {user, 帮我起名}", msgs[0].Role, msgs[0].Content)
	}
}

func TestNamingHandler_PersistsMessages(t *testing.T) {
	store := newAuthTestStore(t)
	seedActiveOrder(t, store, "order-1", "user@example.com", product.ProductNaming, "2027-01-01 00:00:00")

	mockLLM := &agent.MockLLM{
		ToolResps: []*llm.ChatResult{agent.ChatRes(nil, "起名建议如下…")},
	}
	mockTools := &agent.MockToolRegistry{}
	chat := agent.NewAgent(mockLLM, mockTools, "test {locale}\n{phase_instruction}")
	handler := naming.HandleChat(chat, store)

	r := httptest.NewRequest("POST", "/api/naming/agent/chat", strings.NewReader(`{"message":"帮我起名"}`))
	setJWTForTest(t, r, "user@example.com", "order-1")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	msgs, err := store.LoadChatHistory(context.Background(), "order-1")
	if err != nil {
		t.Fatalf("LoadChatHistory: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("message count = %d, want 2 (user + assistant)", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "帮我起名" {
		t.Errorf("msg[0] = {%q, %q}, want {user, 帮我起名}", msgs[0].Role, msgs[0].Content)
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "起名建议如下…" {
		t.Errorf("msg[1] = {%q, %q}, want {assistant, 起名建议如下…}", msgs[1].Role, msgs[1].Content)
	}
}

func TestNamingHandler_LoadsHistory(t *testing.T) {
	store := newAuthTestStore(t)
	seedActiveOrder(t, store, "order-1", "user@example.com", product.ProductNaming, "2027-01-01 00:00:00")

	if err := store.CreateChatMessage(context.Background(), "order-1", string(llm.RoleUser), "我的出生日期是2026-06-25"); err != nil {
		t.Fatalf("seed history: %v", err)
	}
	if err := store.CreateChatMessage(context.Background(), "order-1", string(llm.RoleAssistant), "好的，请确认"); err != nil {
		t.Fatalf("seed history: %v", err)
	}

	mockLLM := &agent.MockLLM{
		ToolResps: []*llm.ChatResult{agent.ChatRes(nil, "确认无误，开始起名")},
	}
	mockTools := &agent.MockToolRegistry{}
	chat := agent.NewAgent(mockLLM, mockTools, "test {locale}\n{phase_instruction}")
	handler := naming.HandleChat(chat, store)

	r := httptest.NewRequest("POST", "/api/naming/agent/chat", strings.NewReader(`{"message":"确认"}`))
	setJWTForTest(t, r, "user@example.com", "order-1")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	// 2 historical + user + assistant = 4.
	msgs, err := store.LoadChatHistory(context.Background(), "order-1")
	if err != nil {
		t.Fatalf("LoadChatHistory: %v", err)
	}
	if len(msgs) != 4 {
		t.Fatalf("message count = %d, want 4", len(msgs))
	}

	// Verify LLM received the history messages.
	if len(mockLLM.Calls) == 0 {
		t.Fatal("LLM was not called")
	}
	var foundHistoryUser, foundHistoryAsst, foundCurrent bool
	for _, m := range mockLLM.Calls[0] {
		if m.Role == llm.RoleUser && m.Content == "我的出生日期是2026-06-25" {
			foundHistoryUser = true
		}
		if m.Role == llm.RoleAssistant && m.Content == "好的，请确认" {
			foundHistoryAsst = true
		}
		if m.Role == llm.RoleUser && m.Content == "确认" {
			foundCurrent = true
		}
	}
	if !foundHistoryUser {
		t.Error("LLM did not receive historical user message")
	}
	if !foundHistoryAsst {
		t.Error("LLM did not receive historical assistant message")
	}
	if !foundCurrent {
		t.Error("LLM did not receive current user message")
	}
}

func setJWTForTest(t *testing.T, r *http.Request, email, orderID string) {
	t.Helper()
	w := httptest.NewRecorder()
	if err := httputil.SetJWTCookie(w, email, orderID); err != nil {
		t.Fatal("SetJWTCookie failed:", err)
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == "liki_token" {
			r.AddCookie(c)
			return
		}
	}
}

// ── chat history handler ──

func TestChatHistory_NoJWT(t *testing.T) {
	store := newAuthTestStore(t)
	handler := naming.HandleChatHistory(store)

	r := httptest.NewRequest("GET", "/api/naming/orders/order-1/messages", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestChatHistory_CrossOrder(t *testing.T) {
	store := newAuthTestStore(t)
	seedActiveOrder(t, store, "order-1", "user@example.com", product.ProductNaming, "2027-01-01 00:00:00")
	handler := naming.HandleChatHistory(store)

	// JWT is for order-1, but URL path is order-2.
	r := httptest.NewRequest("GET", "/api/naming/orders/order-2/messages", nil)
	r.SetPathValue("id", "order-2")
	setJWTForTest(t, r, "user@example.com", "order-1")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestChatHistory_Empty(t *testing.T) {
	store := newAuthTestStore(t)
	seedActiveOrder(t, store, "order-1", "user@example.com", product.ProductNaming, "2027-01-01 00:00:00")
	handler := naming.HandleChatHistory(store)

	r := httptest.NewRequest("GET", "/api/naming/orders/order-1/messages", nil)
	r.SetPathValue("id", "order-1")
	setJWTForTest(t, r, "user@example.com", "order-1")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var env envelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	msgs, ok := env.Data.([]any)
	if !ok {
		t.Fatalf("data is not an array: %T", env.Data)
	}
	if len(msgs) != 0 {
		t.Errorf("len = %d, want 0", len(msgs))
	}
}

func TestChatHistory_WithMessages(t *testing.T) {
	store := newAuthTestStore(t)
	seedActiveOrder(t, store, "order-1", "user@example.com", product.ProductNaming, "2027-01-01 00:00:00")

	if err := store.CreateChatMessage(context.Background(), "order-1", string(llm.RoleUser), "我的姓氏是陈"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := store.CreateChatMessage(context.Background(), "order-1", string(llm.RoleAssistant), "好的，请提供出生日期"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	handler := naming.HandleChatHistory(store)

	r := httptest.NewRequest("GET", "/api/naming/orders/order-1/messages", nil)
	r.SetPathValue("id", "order-1")
	setJWTForTest(t, r, "user@example.com", "order-1")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var env envelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	// envelope.Data is []any after JSON decode.
	arr, ok := env.Data.([]any)
	if !ok {
		t.Fatalf("data is not an array: %T", env.Data)
	}
	if len(arr) != 2 {
		t.Fatalf("len = %d, want 2", len(arr))
	}

	m0 := arr[0].(map[string]any)
	if m0["role"] != "user" || m0["content"] != "我的姓氏是陈" {
		t.Errorf("msg[0] = {%v, %v}", m0["role"], m0["content"])
	}
	if m0["created_at"] == nil || m0["created_at"].(string) == "" {
		t.Error("msg[0].created_at is empty")
	}

	m1 := arr[1].(map[string]any)
	if m1["role"] != "assistant" || m1["content"] != "好的，请提供出生日期" {
		t.Errorf("msg[1] = {%v, %v}", m1["role"], m1["content"])
	}
	if m1["created_at"] == nil || m1["created_at"].(string) == "" {
		t.Error("msg[1].created_at is empty")
	}
}
