package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestToolSchemasMatchDeepSeekRequirements(t *testing.T) {
	t.Parallel()
	// Serve a minimal rpc.discover response so NewToolRegistry can
	// initialise without a running engine.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"result": map[string]any{
				"methods": func() []map[string]any {
					methods := []map[string]any{}
					for _, name := range []string{
						"city", "tianwen.time", "bazi.chart", "ziwei.chart",
						"qiming.pick", "qiming.build", "qiming.check",
					} {
						methods = append(methods, map[string]any{
							"name":        name,
							"description": name,
							"params":      map[string]any{"type": "object", "properties": map[string]any{}},
						})
					}
					return methods
				}(),
			},
			"id": 1,
		}); err != nil {
			panic(err)
		}
	}))
	defer srv.Close()

	reg := NewToolRegistry(srv.URL)
	schemas := reg.Schemas()

	if len(schemas) == 0 {
		t.Fatal("no tool schemas registered")
	}

	namePattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

	for i, s := range schemas {
		if s.Type != "function" {
			t.Errorf("schema[%d].type = %q, want 'function'", i, s.Type)
		}

		var fn struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			Parameters  json.RawMessage `json:"parameters,omitempty"`
		}
		if err := json.Unmarshal(s.Function, &fn); err != nil {
			t.Errorf("schema[%d] invalid JSON: %v", i, err)
			continue
		}

		if !namePattern.MatchString(fn.Name) {
			t.Errorf("tool %q name does not match ^[a-zA-Z0-9_-]+$", fn.Name)
		}

		if len(fn.Parameters) > 0 {
			var params struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(fn.Parameters, &params); err != nil {
				t.Errorf("tool %q parameters invalid JSON: %v", fn.Name, err)
			}
			if params.Type != "" && params.Type != "object" {
				t.Errorf("tool %q parameters.type = %q, want 'object'", fn.Name, params.Type)
			}
		}
	}

	t.Logf("verified %d tool schemas comply with DeepSeek requirements", len(schemas))
}

func TestNewToolRegistryWithoutEngine(t *testing.T) {
	t.Parallel()
	// NewToolRegistry must not panic or block when the engine is unreachable.
	reg := NewToolRegistry("http://127.0.0.1:19999/jsonrpc")
	if reg == nil {
		t.Fatal("NewToolRegistry returned nil")
	}
	if reg.handlers == nil {
		t.Fatal("handlers map not initialized")
	}
}

func TestSchemasEmptyWhenEngineDown(t *testing.T) {
	t.Parallel()
	reg := NewToolRegistry("http://127.0.0.1:19999/jsonrpc")
	schemas := reg.Schemas()
	if len(schemas) != 0 {
		t.Errorf("expected empty schemas when engine is down, got %d", len(schemas))
	}
}

func TestExecuteUnknownToolWhenEngineDown(t *testing.T) {
	t.Parallel()
	reg := NewToolRegistry("http://127.0.0.1:19999/jsonrpc")
	_, err := reg.Execute(context.TODO(), "bazi_chart", nil)
	if err == nil {
		t.Fatal("expected error for unknown tool, got nil")
	}
}

func TestReadFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.md"), []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "nested.md"), []byte("world"), 0600); err != nil {
		t.Fatal(err)
	}

	r := NewToolRegistry("")
	r.Register("read_file", "Read a file",
		json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`),
		ReadFileHandler(dir))

	t.Run("valid file", func(t *testing.T) {
		result, err := r.Execute(context.Background(), "read_file", json.RawMessage(`{"path":"test.md"}`))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(result), "hello") {
			t.Errorf("unexpected: %s", result)
		}
	})
	t.Run("nested path", func(t *testing.T) {
		result, err := r.Execute(context.Background(), "read_file", json.RawMessage(`{"path":"sub/nested.md"}`))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(result), "world") {
			t.Errorf("unexpected: %s", result)
		}
	})
	t.Run("directory traversal blocked", func(t *testing.T) {
		_, err := r.Execute(context.Background(), "read_file", json.RawMessage(`{"path":"../etc/passwd"}`))
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("absolute path blocked", func(t *testing.T) {
		_, err := r.Execute(context.Background(), "read_file", json.RawMessage(`{"path":"/etc/passwd"}`))
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("file not found", func(t *testing.T) {
		_, err := r.Execute(context.Background(), "read_file", json.RawMessage(`{"path":"noexist.md"}`))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestDiscoverEngineToolsSucceedsWhenEngineAvailable(t *testing.T) {
	t.Parallel()
	// Serve a minimal rpc.discover response.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"result": map[string]any{
				"methods": []map[string]any{
					{"name": "bazi.chart", "description": "八字排盘", "params": map[string]any{"type": "object"}},
				},
			},
			"id": 1,
		}); err != nil {
			panic(err)
		}
	}))
	defer srv.Close()

	reg := NewToolRegistry(srv.URL)

	// Before first use, tools are empty
	if len(reg.Schemas()) == 0 {
		t.Fatal("expected tools after lazy init")
	}

	// Schemas() is idempotent
	s1 := reg.Schemas()
	s2 := reg.Schemas()
	if len(s1) != len(s2) {
		t.Fatalf("Schemas not idempotent: %d vs %d", len(s1), len(s2))
	}
}
