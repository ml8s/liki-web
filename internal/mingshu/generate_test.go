package mingshu

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"liki-web/internal/agent"
	"liki-web/internal/llm"
	"liki-web/internal/product"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	return NewStore(db)
}

func seedLifeBookOrder(t *testing.T, store *Store, orderID, birth string) {
	t.Helper()
	store.CreateOrder(context.Background(), orderID, string(product.ProductMingShu), 12900, "CNY", "test@t.com", birth, "zh-Hans") //nolint:errcheck
}

func TestGenerator_Generate_ValidJSON(t *testing.T) {
	t.Parallel()
	reportJSON := `{"sections":{"personality":{"title":"性格","data":"己土","analysis":"温厚","advice":"宜"},"career":{"title":"事业","data":"金","analysis":"财运旺","advice":"宜"},"love":{"title":"情感","data":"桃花","analysis":"顺利","advice":"宜"},"health":{"title":"健康","data":"五行","analysis":"注意","advice":"不做医学诊断"},"fortune":{"title":"大运","data":"壬辰","analysis":"平稳","advice":"宜","phases":[{"age":"36-45","pillar":"壬辰","analysis":"平","advice":"宜"}]},"milestones":{"title":"关键","data":"节点","analysis":"注意","advice":"宜"}}}`
	mockLLM := &agent.MockLLM{
		ToolResps: []*llm.ChatResult{agent.ChatRes(nil, reportJSON)},
	}
	a := agent.NewAgent(mockLLM, &agent.MockToolRegistry{}, "")
	p := &Generator{agent: a, store: nil}

	result, err := p.generateReport(context.Background(), "Test | 1990-05-20 12:00 | Shanghai | male")
	if err != nil {
		t.Fatal(err)
	}
	var data ReportData
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, result)
	}
	if data.Sections.Personality.Data == "" {
		t.Error("personality data is empty")
	}
}

func TestGenerator_Review_Pass(t *testing.T) {
	t.Parallel()
	evalJSON := `{"pass":true,"feedback":{}}`
	mockLLM := &agent.MockLLM{
		ToolResps: []*llm.ChatResult{agent.ChatRes(nil, evalJSON)},
	}
	a := agent.NewAgent(mockLLM, &agent.MockToolRegistry{}, "")
	p := &Generator{agent: a, store: nil}

	eval, err := p.reviewReport(context.Background(), "dummy report")
	if err != nil {
		t.Fatal(err)
	}
	if !eval.Pass {
		t.Error("expected pass=true")
	}
}

func TestGenerator_Review_Fail(t *testing.T) {
	t.Parallel()
	evalJSON := `{"pass":false,"feedback":{"personality":"too short","career":"missing data"}}`
	mockLLM := &agent.MockLLM{
		ToolResps: []*llm.ChatResult{agent.ChatRes(nil, evalJSON)},
	}
	a := agent.NewAgent(mockLLM, &agent.MockToolRegistry{}, "")
	p := &Generator{agent: a, store: nil}

	eval, err := p.reviewReport(context.Background(), "dummy report")
	if err != nil {
		t.Fatal(err)
	}
	if eval.Pass {
		t.Error("expected pass=false")
	}
	if eval.Feedback["personality"] == "" {
		t.Error("expected feedback for personality")
	}
}

func TestGenerator_Review_BadJSON(t *testing.T) {
	t.Parallel()
	mockLLM := &agent.MockLLM{
		ToolResps: []*llm.ChatResult{agent.ChatRes(nil, "not json at all")},
	}
	a := agent.NewAgent(mockLLM, &agent.MockToolRegistry{}, "")
	p := &Generator{agent: a, store: nil}

	eval, err := p.reviewReport(context.Background(), "dummy report")
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if eval.Pass {
		t.Error("expected pass=false")
	}
}

func TestGenerator_Revise_WithFeedback(t *testing.T) {
	t.Parallel()
	fixedJSON := `{"sections":{"personality":{"title":"性格","data":"己土","analysis":"温厚守信","advice":"宜"}}}`
	mockLLM := &agent.MockLLM{
		ToolResps: []*llm.ChatResult{agent.ChatRes(nil, fixedJSON)},
	}
	a := agent.NewAgent(mockLLM, &agent.MockToolRegistry{}, "")
	p := &Generator{agent: a, store: nil}

	result, err := p.generateReportWithFeedback(context.Background(),
		"Test | 1990-05-20 12:00 | Shanghai | male",
		"old report",
		map[string]string{"personality": "too short"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "己土") {
		t.Errorf("expected fixed content in result, got: %s", result)
	}
}

func TestGenerator_Run_Integration(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedLifeBookOrder(t, store, "test-run", "Tester | 1990-05-20 12:00 | Shanghai | male")

	reportJSON := `{"sections":{"personality":{"title":"性格","data":"己土","analysis":"温厚","advice":"宜"},"career":{"title":"事业","data":"金","analysis":"财运旺","advice":"宜"},"love":{"title":"情感","data":"桃花","analysis":"顺利","advice":"宜"},"health":{"title":"健康","data":"五行","analysis":"注意","advice":"不做医学诊断"},"fortune":{"title":"大运","data":"壬辰","analysis":"平稳","advice":"宜","phases":[{"age":"36-45","pillar":"壬辰","analysis":"平","advice":"宜"}]},"milestones":{"title":"关键","data":"节点","analysis":"注意","advice":"宜"}}}`
	evalJSON := `{"pass":true,"feedback":{}}`

	mockLLM := &agent.MockLLM{
		ToolResps: []*llm.ChatResult{
			agent.ChatRes(nil, reportJSON),
			agent.ChatRes(nil, evalJSON),
		},
	}
	a := agent.NewAgent(mockLLM, &agent.MockToolRegistry{}, "")
	p := &Generator{agent: a, store: store}

	err := p.Run(context.Background(), "test-run")
	if err != nil {
		t.Fatal(err)
	}

	content, err := store.GetReport(context.Background(), "test-run")
	if err != nil {
		t.Fatal(err)
	}
	// Verify stored as JSON
	var data ReportData
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		t.Fatalf("stored content is not valid JSON: %v", err)
	}
	if data.Sections.Personality.Data == "" {
		t.Error("personality data missing in stored JSON")
	}
}

func TestGenerator_ExtractJSON(t *testing.T) {
	t.Parallel()
	tests := []struct{ in, want string }{
		{`some text {"key":"value"} more text`, `{"key":"value"}`},
		{`{"a":1}`, `{"a":1}`},
		{`no json`, ``},
		{`nested { "a": { "b": 1 } }`, `{ "a": { "b": 1 } }`},
	}
	for _, tc := range tests {
		got := extractJSON(tc.in)
		if got != tc.want {
			t.Errorf("extractJSON(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestGenerator_Loop_ReviewFailThenPass(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedLifeBookOrder(t, store, "test-loop", "Tester | 1990-05-20 12:00 | Shanghai | male")

	reportJSON := `{"sections":{"personality":{"title":"性格","data":"己土","analysis":"温厚","advice":"宜"},"career":{"title":"事业","data":"金","analysis":"财运旺","advice":"宜"},"love":{"title":"情感","data":"桃花","analysis":"顺利","advice":"宜"},"health":{"title":"健康","data":"五行","analysis":"注意","advice":"不做医学诊断"},"fortune":{"title":"大运","data":"壬辰","analysis":"平稳","advice":"宜","phases":[{"age":"36-45","pillar":"壬辰","analysis":"平","advice":"宜"}]},"milestones":{"title":"关键","data":"节点","analysis":"注意","advice":"宜"}}}`
	reviewFail := `{"pass":false,"feedback":{"personality":"too short"}}`
	reviewPass := `{"pass":true,"feedback":{}}` //nolint:gosec
	fixedJSON := `{"sections":{"personality":{"title":"性格","data":"己土","analysis":"温厚守信","advice":"宜从事教育"},"career":{"title":"事业","data":"金","analysis":"财运旺","advice":"宜"},"love":{"title":"情感","data":"桃花","analysis":"顺利","advice":"宜"},"health":{"title":"健康","data":"五行","analysis":"注意","advice":"不做医学诊断"},"fortune":{"title":"大运","data":"壬辰","analysis":"平稳","advice":"宜","phases":[{"age":"36-45","pillar":"壬辰","analysis":"平","advice":"宜"}]},"milestones":{"title":"关键","data":"节点","analysis":"注意","advice":"宜"}}}`

	mockLLM := &agent.MockLLM{
		ToolResps: []*llm.ChatResult{
			agent.ChatRes(nil, reportJSON), // generate
			agent.ChatRes(nil, reviewFail), // review → fail
			agent.ChatRes(nil, fixedJSON),  // revise
			agent.ChatRes(nil, reviewPass), // review → pass
		},
	}
	a := agent.NewAgent(mockLLM, &agent.MockToolRegistry{}, "")
	p := &Generator{agent: a, store: store}

	err := p.Run(context.Background(), "test-loop")
	if err != nil {
		t.Fatal(err)
	}

	content, err := store.GetReport(context.Background(), "test-loop")
	if err != nil {
		t.Fatal(err)
	}
	// The final stored report should be the fixed one
	if !strings.Contains(content, "教育") {
		t.Errorf("expected fixed content, got: %s", content)
	}
}

func TestGenerator_BirthNotFound(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	mockLLM := &agent.MockLLM{
		ToolResps: []*llm.ChatResult{agent.ChatRes(nil, "{}")},
	}
	a := agent.NewAgent(mockLLM, &agent.MockToolRegistry{}, "")
	p := &Generator{agent: a, store: store}

	err := p.Run(context.Background(), "no-such-order")
	if err == nil {
		t.Fatal("expected error for missing birth")
	}
}

func TestGenerator_Process_Helper(t *testing.T) {
	t.Parallel()
	mockLLM := &agent.MockLLM{
		ToolResps: []*llm.ChatResult{agent.ChatRes(nil, "hello world")},
	}
	a := agent.NewAgent(mockLLM, &agent.MockToolRegistry{}, "")
	p := &Generator{agent: a, store: nil}

	result, err := p.process(context.Background(), "system", "user")
	if err != nil {
		t.Fatal(err)
	}
	if result != "hello world" {
		t.Errorf("expected 'hello world', got %q", result)
	}
}

func TestGenerator_JSONMapString(t *testing.T) {
	tests := []struct {
		name   string
		input  map[string]string
		expect string
	}{
		{"non-empty", map[string]string{"title": "too short", "analysis": "add more"}, `"add more"`},
		{"nil", nil, ""},
		{"empty", map[string]string{}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := jsonMapString(tc.input)
			if !strings.Contains(got, tc.expect) && (tc.expect != "" || got != "") {
				t.Errorf("jsonMapString(%v) = %q, want %q", tc.input, got, tc.expect)
			}
		})
	}
}

func TestGenerator_Run_MaxLoop(t *testing.T) {
	s := newTestStore(t)
	oid := uuid.New().String()
	seedLifeBookOrder(t, s, oid, "1990-01-01")

	mockLLM := &agent.MockLLM{
		ToolResps: []*llm.ChatResult{
			agent.ChatRes(nil, `{"title":"r0","sections":{"a":{"title":"a","analysis":"b"}}}`), // 0 generate
			agent.ChatRes(nil, `{"pass":false,"feedback":{"a":"fix"}}`),                        // 1 initial review
			agent.ChatRes(nil, `{"title":"r1","sections":{"a":{"title":"a","analysis":"b"}}}`), // 2 revise 0
			agent.ChatRes(nil, `{"pass":false,"feedback":{"b":"fix2"}}`),                       // 3 review 0
			agent.ChatRes(nil, `{"title":"r2","sections":{"a":{"title":"a","analysis":"b"}}}`), // 4 revise 1
			agent.ChatRes(nil, `{"pass":false,"feedback":{"c":"fix3"}}`),                       // 5 review 1
			agent.ChatRes(nil, `{"title":"r3","sections":{"a":{"title":"a","analysis":"b"}}}`), // 6 revise 2
			agent.ChatRes(nil, `{"pass":true}`),                                                // 7 review 2 (still runs; loop exits after this round)
		},
	}
	mockTools := &agent.MockToolRegistry{}
	chat := agent.NewAgent(mockLLM, mockTools, "prompt")

	g := &Generator{agent: chat, store: s}
	g.generatePrompt = "gen"
	g.reviewPrompt = "review"
	g.revisePrompt = "revise"

	err := g.Run(context.Background(), oid)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	content, _ := s.GetReport(context.Background(), oid) //nolint:errcheck
	if !strings.Contains(content, "r3") {
		t.Errorf("expected last revise content, got %q", content)
	}
}
