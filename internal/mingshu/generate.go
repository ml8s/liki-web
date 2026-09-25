package mingshu

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ml8s/liki-web/internal/agent"
	"github.com/ml8s/liki-web/internal/llm"
)

// Generator runs the dual-agent mingshu generation loop.
type Generator struct {
	agent          *agent.Agent
	store          *Store
	generatePrompt string
	reviewPrompt   string
	revisePrompt   string
}

// NewGenerator creates a Generator.
func NewGenerator(agent *agent.Agent, store *Store) *Generator {
	return &Generator{agent: agent, store: store}
}

// LoadPromptsFrom reads prompts from a skills directory.
func (g *Generator) LoadPromptsFrom(skillsDir string) error {
	for name, dst := range map[string]*string{
		"generate.md": &g.generatePrompt,
		"review.md":   &g.reviewPrompt,
		"revise.md":   &g.revisePrompt,
	} {
		data, err := os.ReadFile(filepath.Join(skillsDir, name))
		if err != nil {
			return fmt.Errorf("mingshu: read %s: %w", name, err)
		}
		*dst = string(data)
	}
	return nil
}

func (g *Generator) process(ctx context.Context, systemPrompt, userContent string) (string, error) {
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: systemPrompt},
		{Role: llm.RoleUser, Content: userContent},
	}
	result, err := g.agent.Execute(ctx, msgs, nil, nil)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for i := len(result) - 1; i >= 0; i-- {
		if result[i].Role == llm.RoleAssistant && result[i].Content != "" {
			b.WriteString(result[i].Content)
		}
	}
	if b.Len() == 0 {
		return "", fmt.Errorf("agent returned empty response")
	}
	return b.String(), nil
}

// Run executes the full dual-agent pipeline for an order.
func (g *Generator) Run(ctx context.Context, orderID string) error {
	birth, err := g.store.GetBirth(ctx, orderID)
	if err != nil {
		return fmt.Errorf("mingshu: get birth: %w", err)
	}

	reportJSON, err := g.generateReport(ctx, birth)
	if err != nil {
		return err
	}

	eval, err := g.reviewReport(ctx, reportJSON)
	if err != nil {
		return err
	}

	for round := 0; round < 3 && !eval.Pass; round++ {
		fixedReport, err := g.generateReportWithFeedback(ctx, birth, reportJSON, eval.Feedback)
		if err != nil {
			return err
		}
		reportJSON = fixedReport
		eval, err = g.reviewReport(ctx, reportJSON)
		if err != nil {
			return err
		}
	}

	return g.store.SaveReport(ctx, orderID, reportJSON)
}

func (g *Generator) generateReport(ctx context.Context, birth string) (string, error) {
	result, err := g.process(ctx, g.generatePrompt, birth)
	if err != nil {
		return "", fmt.Errorf("mingshu: generate: %w", err)
	}
	result = extractJSON(result)
	if !json.Valid([]byte(result)) {
		return "", fmt.Errorf("mingshu: generate: invalid JSON after extraction")
	}
	return result, nil
}

func (g *Generator) reviewReport(ctx context.Context, reportJSON string) (evalResult, error) {
	result, err := g.process(ctx, g.reviewPrompt, reportJSON)
	if err != nil {
		return evalResult{}, fmt.Errorf("mingshu: review: %w", err)
	}
	result = extractJSON(result)
	var e evalResult
	if err := json.Unmarshal([]byte(result), &e); err != nil {
		return evalResult{Pass: false, Feedback: map[string]string{"_": result}}, nil //nolint:nilerr
	}
	return e, nil
}

func (g *Generator) generateReportWithFeedback(ctx context.Context, birth, reportJSON string, feedback map[string]string) (string, error) {
	input := fmt.Sprintf("出生信息: %s\n\n现有报告:\n%s\n\n审查反馈:\n%s", birth, reportJSON, jsonMapString(feedback))
	result, err := g.process(ctx, g.revisePrompt, input)
	if err != nil {
		return "", fmt.Errorf("mingshu: revise: %w", err)
	}
	return result, nil
}

type evalResult struct {
	Pass     bool              `json:"pass"`
	Feedback map[string]string `json:"feedback"`
}

func extractJSON(s string) string {
	start := strings.Index(s, "{")
	if start < 0 {
		return ""
	}
	end := strings.LastIndex(s, "}")
	if end < 0 || end <= start {
		return s
	}
	return s[start : end+1]
}

func jsonMapString(m map[string]string) string {
	if len(m) == 0 {
		return "{}"
	}
	var b strings.Builder
	b.WriteByte('{')
	first := true
	for k, v := range m {
		if !first {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%q: %q", k, v)
		first = false
	}
	b.WriteByte('}')
	return b.String()
}
