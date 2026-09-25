package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ml8s/liki-web/internal/llm"
)

// ToolScope is the explicit authority granted to one chat skill. Exact names
// cover built-in tools; prefixes cover engine domain methods.
type ToolScope struct {
	Exact    []string
	Prefixes []string
}

// SkillToolScopes are the backend security boundary for unified chat. The LLM
// neither chooses its authority nor sees schemas outside the active skill. The
// unified "liki" scope is the union of the four domains (本命/起名/问卦/风水);
// domain routing happens in the prompt, not by widening authority per turn.
var SkillToolScopes = map[string]ToolScope{
	"liki": {
		Exact: []string{
			"read_file", "update_profile_birth_time",
			// naming RPC methods without a domain prefix
			"city.coords",
			"qiming.surname", "qiming.pick", "qiming.compose", "qiming.check", "qiming.char",
		},
		Prefixes: []string{
			"bazi.", "ziwei.", "liuyao.", "qimen.", "huangli.", "tianwen.", "bazhai.", "xuankong.",
		},
	},
}

// ScopedToolRegistry wraps a registry with deny-by-default tool access.
type ScopedToolRegistry struct {
	base     ToolRegistry
	scope    ToolScope
	exact    map[string]bool
	prefixes []string
}

func NewScopedToolRegistry(base ToolRegistry, scope ToolScope) *ScopedToolRegistry {
	exact := make(map[string]bool, len(scope.Exact))
	for _, name := range scope.Exact {
		exact[name] = true
	}
	return &ScopedToolRegistry{base: base, scope: scope, exact: exact, prefixes: scope.Prefixes}
}

func (r *ScopedToolRegistry) allowed(name string) bool {
	if r.exact[name] {
		return true
	}
	for _, prefix := range r.prefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func (r *ScopedToolRegistry) Schemas() []llm.ToolDef {
	schemas := r.base.Schemas()
	filtered := make([]llm.ToolDef, 0, len(schemas))
	for _, schema := range schemas {
		var def struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(schema.Function, &def); err == nil && r.allowed(def.Name) {
			filtered = append(filtered, schema)
		}
	}
	return filtered
}

func (r *ScopedToolRegistry) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	if !r.allowed(name) {
		return nil, fmt.Errorf("tool %q is not allowed for this skill", name)
	}
	return r.base.Execute(ctx, name, args)
}
