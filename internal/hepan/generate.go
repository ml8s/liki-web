package hepan

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

const userTemplate = "请为以下两人进行八紫合盘分析：\n\n第一人：%s\n出生时间：%s\n出生地点：%s\n性别：%s\n\n第二人：%s\n出生时间：%s\n出生地点：%s\n性别：%s\n\n请使用命理工具获取排盘和合盘数据，然后生成综合报告。"

// Generator runs the hepan report pipeline.
type Generator struct {
	agent          *agent.Agent
	store          *Store
	generatePrompt string
	reviewPrompt   string
	revisePrompt   string
}

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
			return fmt.Errorf("hepan: read %s: %w", name, err)
		}
		*dst = string(data)
	}
	return nil
}

// Run executes the full pipeline for an order.
func (g *Generator) Run(ctx context.Context, orderID string) error {
	order, err := g.store.GetOrder(ctx, orderID)
	if err != nil {
		return fmt.Errorf("hepan: get order: %w", err)
	}

	var a, b PersonInfo
	_ = json.Unmarshal([]byte(order.PersonA), &a) //nolint:errcheck
	_ = json.Unmarshal([]byte(order.PersonB), &b) //nolint:errcheck

	reportJSON, err := g.generateReport(ctx, a, b)
	if err != nil {
		return err
	}

	eval, err := g.reviewReport(ctx, reportJSON)
	if err != nil {
		return err
	}

	for round := 0; round < 3 && !eval.Pass; round++ {
		fixedReport, err := g.generateWithFeedback(ctx, a, b, reportJSON, eval.Feedback)
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

func (g *Generator) generateReport(ctx context.Context, a, b PersonInfo) (string, error) {
	input := fmt.Sprintf(userTemplate, a.Name, a.Birth, a.City, a.Gender, b.Name, b.Birth, b.City, b.Gender)
	result, err := g.process(ctx, g.generatePrompt, input)
	if err != nil {
		return "", fmt.Errorf("hepan: generate: %w", err)
	}
	if !json.Valid([]byte(result)) {
		return "", fmt.Errorf("hepan: generate: invalid JSON")
	}
	return result, nil
}

func (g *Generator) reviewReport(ctx context.Context, reportJSON string) (evalResult, error) {
	result, err := g.process(ctx, g.reviewPrompt, reportJSON)
	if err != nil {
		return evalResult{}, fmt.Errorf("hepan: review: %w", err)
	}
	var e evalResult
	if err := json.Unmarshal([]byte(result), &e); err != nil {
		return evalResult{Pass: false, Feedback: map[string]string{"_": result}}, nil //nolint:nilerr
	}
	return e, nil
}

func (g *Generator) generateWithFeedback(ctx context.Context, a, b PersonInfo, reportJSON string, feedback map[string]string) (string, error) {
	input := fmt.Sprintf(
		"第一人: %s | %s | %s | %s\n第二人: %s | %s | %s | %s\n\n现有报告:\n%s\n\n审查反馈:\n%s",
		a.Name, a.Birth, a.City, a.Gender, b.Name, b.Birth, b.City, b.Gender,
		reportJSON, jsonMapString(feedback))
	result, err := g.process(ctx, g.revisePrompt, input)
	if err != nil {
		return "", fmt.Errorf("hepan: revise: %w", err)
	}
	return result, nil
}

func (g *Generator) process(ctx context.Context, systemPrompt, userContent string) (string, error) {
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: systemPrompt},
		{Role: llm.RoleUser, Content: userContent},
	}
	result, err := g.agent.Execute(ctx, msgs, func(agent.ChatEvent) {}, nil)
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
	return extractJSON(b.String()), nil
}

// evalResult is the JSON structure for review output.
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
