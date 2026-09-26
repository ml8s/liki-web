package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ml8s/liki-web/internal/llm"
)

type fakeScopedRegistry struct {
	executed string
}

func (r *fakeScopedRegistry) Schemas() []llm.ToolDef {
	return []llm.ToolDef{
		{Type: llm.ToolTypeFunction, Function: json.RawMessage(`{"name":"allowed.tool"}`)},
		{Type: llm.ToolTypeFunction, Function: json.RawMessage(`{"name":"forbidden.tool"}`)},
	}
}

func (r *fakeScopedRegistry) Execute(_ context.Context, name string, _ json.RawMessage) (json.RawMessage, error) {
	r.executed = name
	return json.RawMessage(`{"ok":true}`), nil
}

func TestScopedToolRegistryFiltersSchemasAndExecution(t *testing.T) {
	base := &fakeScopedRegistry{}
	scoped := NewScopedToolRegistry(base, ToolScope{Exact: []string{"allowed.exact"}, Prefixes: []string{"allowed."}})

	schemas := scoped.Schemas()
	if len(schemas) != 1 {
		t.Fatalf("schemas = %d, want 1", len(schemas))
	}
	var def struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(schemas[0].Function, &def); err != nil {
		t.Fatal(err)
	}
	if def.Name != "allowed.tool" {
		t.Fatalf("schema = %q", def.Name)
	}

	if _, err := scoped.Execute(context.Background(), "forbidden.tool", nil); err == nil {
		t.Fatal("expected forbidden tool execution to fail")
	}
	if base.executed != "" {
		t.Fatalf("base executed forbidden tool %q", base.executed)
	}
	if _, err := scoped.Execute(context.Background(), "allowed.exact", nil); err != nil {
		t.Fatal(err)
	}
	if base.executed != "allowed.exact" {
		t.Fatalf("base executed = %q", base.executed)
	}
}

func TestBuildScopedChatPromptIsIsolated(t *testing.T) {
	dir := t.TempDir()
	if err := mkdirAllAndWrite(dir, "liki", "skill"); err != nil {
		t.Fatal(err)
	}
	if err := mkdirAllAndWrite(dir, "liki/webapp", "secret"); err != nil {
		t.Fatal(err)
	}

	prompt, err := BuildScopedChatPrompt(dir, "liki")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "Liki Chat") {
		t.Fatalf("prompt missing Liki Chat wording: %q", prompt)
	}
	if _, err := BuildScopedChatPrompt(dir, "unknown-skill"); err == nil {
		t.Fatal("expected legacy skill id to fail")
	}
	if _, err := BuildScopedChatPrompt(dir, "../liki"); err == nil {
		t.Fatal("expected invalid skill id to fail")
	}
}

func mkdirAllAndWrite(root, relative, content string) error {
	path := filepath.Join(root, relative, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}

func TestLikiChatScopeAllowsEngineDomainTools(t *testing.T) {
	scope := SkillToolScopes["liki"]
	base := &fakeScopedRegistry{}
	scoped := NewScopedToolRegistry(base, scope)

	for _, name := range []string{
		"bazi.chart", "ziwei.fullchart", "liuyao.snapshot", "qimen.ask",
		"huangli.days", "tianwen.time", "bazhai.layout",
	} {
		if _, err := scoped.Execute(context.Background(), name, json.RawMessage(`{}`)); err != nil {
			t.Fatalf("engine tool %q rejected: %v", name, err)
		}
	}

	if _, err := scoped.Execute(context.Background(), "full_paipan", json.RawMessage(`{}`)); err == nil {
		t.Fatal("legacy full_paipan should no longer be registered/allowed")
	}
}
