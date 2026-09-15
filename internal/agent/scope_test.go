package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"liki-web/internal/llm"
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
	if !strings.Contains(prompt, "统一命理 skill") {
		t.Fatalf("prompt missing unified wording: %q", prompt)
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
