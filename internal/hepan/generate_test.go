package hepan

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"liki-web/internal/agent"
	"liki-web/internal/llm"
)

func TestLoadPromptsFrom_LoadsAllFiles(t *testing.T) {
	// Use existing prompts from web/skills/webapp/hepan/
	g := &Generator{}
	err := g.LoadPromptsFrom("../../web/skills/liki/webapp/hepan")
	if err != nil {
		t.Fatalf("LoadPromptsFrom: %v", err)
	}
	if g.generatePrompt == "" {
		t.Error("generatePrompt not loaded")
	}
	if g.reviewPrompt == "" {
		t.Error("reviewPrompt not loaded")
	}
	if g.revisePrompt == "" {
		t.Error("revisePrompt not loaded")
	}
}

func TestLoadPromptsFrom_MissingDir(t *testing.T) {
	g := &Generator{}
	err := g.LoadPromptsFrom("/nonexistent/hepan")
	if err == nil {
		t.Fatal("expected error for missing dir")
	}
}

func generateSeed(t *testing.T, store *Store) string {
	t.Helper()
	oid := uuid.New().String()
	pA, _ := json.Marshal(PersonInfo{Name: "张三", Birth: "1990-05-20", City: "上海", Gender: "male"}) //nolint:errcheck
	pB, _ := json.Marshal(PersonInfo{Name: "李四", Birth: "1992-08-15", City: "北京", Gender: "female"}) //nolint:errcheck
	store.CreateOrder(context.Background(), oid, "hepan", 19900, "CNY", "t@t.com", string(pA), string(pB), "zh-Hans") //nolint:errcheck
	return oid
}

func TestGenerator_Run_GenerateOnly(t *testing.T) {
	s := newTestStore(t)
	oid := generateSeed(t, s)

	// Mock LLM: one generate response, review passes
	mockLLM := &agent.MockLLM{
		ToolResps: []*llm.ChatResult{
			agent.ChatRes(nil, `{"title":"test","sections":[{"title":"x","analysis":"y"}]}`),
		},
	}
	mockTools := &agent.MockToolRegistry{}
	chat := agent.NewAgent(mockLLM, mockTools, "prompt")

	g := &Generator{agent: chat, store: s}
	g.generatePrompt = "generate"
	g.reviewPrompt = "review"
	g.revisePrompt = "revise"

	// Mock the review prompt to return pass=true
	mockLLM.ToolResps = append(mockLLM.ToolResps,
		agent.ChatRes(nil, `{"pass":true,"feedback":{}}`))

	err := g.Run(context.Background(), oid)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify report was saved
	content, err := s.GetReport(context.Background(), oid)
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}
	if !strings.Contains(content, "test") {
		t.Errorf("report = %q, want title=test", content)
	}
}

func TestGenerator_Run_ReviewThenRevise(t *testing.T) {
	s := newTestStore(t)
	oid := generateSeed(t, s)

	mockLLM := &agent.MockLLM{
		ToolResps: []*llm.ChatResult{
			agent.ChatRes(nil, `{"title":"v1","sections":[{"title":"x","analysis":"y"}]}`),
			agent.ChatRes(nil, `{"pass":false,"feedback":{"x":"too short"}}`),
			agent.ChatRes(nil, `{"title":"v2","sections":[{"title":"x","analysis":"longer"}]}`),
			agent.ChatRes(nil, `{"pass":true,"feedback":{}}`),
		},
	}
	mockTools := &agent.MockToolRegistry{}
	chat := agent.NewAgent(mockLLM, mockTools, "prompt")

	g := &Generator{agent: chat, store: s}
	g.generatePrompt = "generate"
	g.reviewPrompt = "review"
	g.revisePrompt = "revise"

	err := g.Run(context.Background(), oid)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	content, _ := s.GetReport(context.Background(), oid) //nolint:errcheck
	if !strings.Contains(content, "v2") {
		t.Errorf("report = %q, want v2", content)
	}
}

func TestGenerator_Run_EmptyLLMResponse(t *testing.T) {
	s := newTestStore(t)
	oid := generateSeed(t, s)

	mockLLM := &agent.MockLLM{
		ToolResps: []*llm.ChatResult{
			agent.ChatRes(nil, ""),
		},
	}
	mockTools := &agent.MockToolRegistry{}
	chat := agent.NewAgent(mockLLM, mockTools, "prompt")

	g := &Generator{agent: chat, store: s}
	g.generatePrompt = "gen"

	err := g.Run(context.Background(), oid)
	if err == nil {
		t.Fatal("expected error for empty LLM response")
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct{ in, want string }{
		{`some text {"key":"value"} more text`, `{"key":"value"}`},
		{`{"a":1}`, `{"a":1}`},
		{`no json`, ``},
		{`leading {"x":1} trailing`, `{"x":1}`},
	}
	for _, tc := range tests {
		got := extractJSON(tc.in)
		if got != tc.want {
			t.Errorf("extractJSON(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestJSONMapString(t *testing.T) {
	m := map[string]string{"a": "1", "b": "2"}
	s := jsonMapString(m)
	if !strings.Contains(s, `"a": "1"`) || !strings.Contains(s, `"b": "2"`) {
		t.Errorf("jsonMapString = %q", s)
	}
	if s == "{}" {
		t.Error("expected non-empty map")
	}
	// Empty map
	s2 := jsonMapString(nil)
	if s2 != "{}" {
		t.Errorf("empty = %q, want {}", s2)
	}
}


func TestGenerator_Review_Pass(t *testing.T) {
  mockLLM := &agent.MockLLM{
    ToolResps: []*llm.ChatResult{agent.ChatRes(nil, `{"pass":true}`)},
  }
  g := &Generator{agent: agent.NewAgent(mockLLM, &agent.MockToolRegistry{}, "prompt")}
  g.reviewPrompt = "review"

  result, err := g.reviewReport(context.Background(), `{"title":"test"}`)
  if err != nil { t.Fatalf("reviewReport: %v", err) }
  if !result.Pass { t.Error("expected pass=true") }
}

func TestGenerator_Review_Fail(t *testing.T) {
  mockLLM := &agent.MockLLM{
    ToolResps: []*llm.ChatResult{agent.ChatRes(nil, `{"pass":false,"feedback":{"title":"too short"}}`)},
  }
  g := &Generator{agent: agent.NewAgent(mockLLM, &agent.MockToolRegistry{}, "prompt")}
  g.reviewPrompt = "review"

  result, err := g.reviewReport(context.Background(), `{"title":"test"}`)
  if err != nil { t.Fatalf("reviewReport: %v", err) }
  if result.Pass { t.Error("expected pass=false") }
  if result.Feedback["title"] != "too short" { t.Errorf("feedback = %v", result.Feedback) }
}

func TestGenerator_Review_BadJSON(t *testing.T) {
  mockLLM := &agent.MockLLM{
    ToolResps: []*llm.ChatResult{agent.ChatRes(nil, "not json")},
  }
  g := &Generator{agent: agent.NewAgent(mockLLM, &agent.MockToolRegistry{}, "prompt")}
  g.reviewPrompt = "review"

  result, err := g.reviewReport(context.Background(), `{"title":"test"}`)
  if err != nil { t.Fatalf("reviewReport: %v", err) }
  if result.Pass { t.Error("expected pass=false for bad JSON") }
  if _, ok := result.Feedback["_"]; !ok {
    t.Error("expected feedback._ for bad JSON")
  }
}

func TestGenerator_Revise_WithFeedback(t *testing.T) {
  mockLLM := &agent.MockLLM{
    ToolResps: []*llm.ChatResult{agent.ChatRes(nil, `{"title":"revised","sections":{"a":{"title":"x","analysis":"y"}}}`)},
  }
  g := &Generator{agent: agent.NewAgent(mockLLM, &agent.MockToolRegistry{}, "prompt")}
  g.revisePrompt = "revise"

  result, err := g.generateWithFeedback(context.Background(), PersonInfo{}, PersonInfo{},
    `{"title":"old"}`, map[string]string{"title":"too short"})
  if err != nil { t.Fatalf("generateWithFeedback: %v", err) }
  if !strings.Contains(result, "revised") { t.Errorf("result = %q, want revised", result) }
}

func TestGenerator_Generate_ValidJSON(t *testing.T) {
  mockLLM := &agent.MockLLM{
    ToolResps: []*llm.ChatResult{agent.ChatRes(nil, `{"title":"gen","persons":[],"sections":{"x":{"title":"x","analysis":"y"}}}`)},
  }
  g := &Generator{agent: agent.NewAgent(mockLLM, &agent.MockToolRegistry{}, "prompt")}
  g.generatePrompt = "generate"

  result, err := g.generateReport(context.Background(), PersonInfo{}, PersonInfo{})
  if err != nil { t.Fatalf("generateReport: %v", err) }
  if !strings.Contains(result, "gen") { t.Errorf("result = %q, want gen", result) }
}

func TestGenerator_Run_MaxLoop(t *testing.T) {
  s := newTestStore(t)
  oid := generateSeed(t, s)

  // generate → review(fail) → round 0: revise+review → round 1: revise+review → round 2: revise+review(fail) → review after loop
  mockLLM := &agent.MockLLM{
    ToolResps: []*llm.ChatResult{
      agent.ChatRes(nil, `{"title":"report","persons":[],"sections":{"a":{"title":"a","analysis":"b"}}}`), // 0 generate
      agent.ChatRes(nil, `{"pass":false,"feedback":{"a":"fix"}}`),  // 1 initial review
      agent.ChatRes(nil, `{"title":"r1","persons":[],"sections":{"a":{"title":"a","analysis":"b"}}}`),  // 2 revise 0
      agent.ChatRes(nil, `{"pass":false,"feedback":{"b":"fix2"}}`), // 3 review 0
      agent.ChatRes(nil, `{"title":"r2","persons":[],"sections":{"a":{"title":"a","analysis":"b"}}}`),  // 4 revise 1
      agent.ChatRes(nil, `{"pass":false,"feedback":{"c":"fix3"}}`), // 5 review 1
      agent.ChatRes(nil, `{"title":"r3","persons":[],"sections":{"a":{"title":"a","analysis":"b"}}}`),  // 6 revise 2
      agent.ChatRes(nil, `{"pass":true}`), // 7 final review (round=2 completes, loop checks eval.Pass=false first)
    },
  }
  mockTools := &agent.MockToolRegistry{}
  chat := agent.NewAgent(mockLLM, mockTools, "prompt")

  g := &Generator{agent: chat, store: s}
  g.generatePrompt = "gen"
  g.reviewPrompt = "review"
  g.revisePrompt = "revise"

  err := g.Run(context.Background(), oid)
  if err != nil { t.Fatalf("Run: %v", err) }

  content, _ := s.GetReport(context.Background(), oid) //nolint:errcheck
  if !strings.Contains(content, "r3") {
    t.Errorf("expected last revise content, got %q", content)
  }
}
