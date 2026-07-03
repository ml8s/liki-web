// tools.go — LLM Chat Tool Registry for liki_web
// Refactored: calls the engine via HTTP JSON-RPC instead of direct Go function calls.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"liki_web/internal/llm"
)

var (
	toolsOnce sync.Once
	toolsMap  map[string]toolEntry
)

type toolEntry struct {
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

func initTools() {
	var f struct {
		Tools []struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			Parameters  json.RawMessage `json:"parameters"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(ToolsJSON, &f); err != nil {
		panic("initTools: " + err.Error())
	}
	toolsMap = make(map[string]toolEntry, len(f.Tools))
	for _, t := range f.Tools {
		toolsMap[t.Name] = toolEntry{Description: t.Description, Parameters: t.Parameters}
	}
}

// ValidateTools parses tools.json eagerly so configuration errors fail at startup.
func ValidateTools() error {
	var err error
	toolsOnce.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("tools: %v", r)
			}
		}()
		initTools()
	})
	return err
}

func toolParams(name string) (string, json.RawMessage, error) {
	toolsOnce.Do(initTools)
	e, ok := toolsMap[name]
	if !ok {
		return "", nil, fmt.Errorf("tool not found in tools.json: %s", name)
	}
	return e.Description, e.Parameters, nil
}

// toolRPCMethod maps LLM tool names to engine JSON-RPC method names.
var toolRPCMethod = map[string]string{
	"query_city":              "city",
	"compute_time":            "time.now",
	"compute_chart":           "bazi.chart",
	"compute_ziwei":           "ziwei.chart",
	"compute_naming_wuge":     "qiming.wuge",
	"compute_naming_compose":  "qiming.compose",
	"compute_naming_detail":   "qiming.detail",
	"compute_naming_evaluate": "qiming.evaluate",
}

const (
	ToolQueryCity          = "query_city"
	ToolComputeTime        = "compute_time"
	ToolComputeChart       = "compute_chart"
	ToolComputeZiwei       = "compute_ziwei"
	ToolComputeNamingWuge  = "compute_naming_wuge"
	ToolComputeNamingCompose = "compute_naming_compose"
	ToolComputeNamingDetail  = "compute_naming_detail"
	ToolComputeNamingEvaluate = "compute_naming_evaluate"
)

// ChatToolRegistry executes LLM tool calls by forwarding them as JSON-RPC
// requests to the liki engine server.
type ChatToolRegistry struct {
	engineURL string
	client    *http.Client
	handlers  map[string]func(context.Context, json.RawMessage) (json.RawMessage, error)
	defs      []json.RawMessage
}

// NewNamingToolRegistry creates a tool registry that calls the engine via JSON-RPC.
// If engineURL is empty, it reads from the LIKI_ENGINE_URL env var, defaulting to
// "http://localhost:8080/jsonrpc".
func NewNamingToolRegistry(engineURL string) *ChatToolRegistry {
	if engineURL == "" {
		engineURL = os.Getenv("LIKI_ENGINE_URL")
	}
	if engineURL == "" {
		engineURL = "http://localhost:8080/jsonrpc"
	}

	r := &ChatToolRegistry{
		engineURL: engineURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		handlers: map[string]func(context.Context, json.RawMessage) (json.RawMessage, error){},
	}

	for toolName, rpcMethod := range toolRPCMethod {
		name, method := toolName, rpcMethod
		r.handlers[name] = func(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
			return r.callEngine(ctx, method, args)
		}
	}

	for _, name := range []string{
		ToolQueryCity, ToolComputeTime,
		ToolComputeChart, ToolComputeZiwei,
		ToolComputeNamingWuge, ToolComputeNamingCompose,
		ToolComputeNamingDetail, ToolComputeNamingEvaluate,
	} {
		desc, params, err := toolParams(name)
		if err != nil {
			panic("registerTool: " + err.Error())
		}
		r.defs = append(r.defs, toolDef(name, desc, params))
	}

	return r
}

func (r *ChatToolRegistry) callEngine(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("rpc marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.engineURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("rpc request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rpc call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("rpc read: %w", err)
	}

	var rpcResp struct {
		JSONRPC string          `json:"jsonrpc"`
		Result  json.RawMessage `json:"result,omitempty"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
		ID int `json:"id"`
	}
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("rpc parse: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("engine error (%d): %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

func (r *ChatToolRegistry) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	h, ok := r.handlers[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
	return h(ctx, args)
}

func (r *ChatToolRegistry) Schemas() []llm.ToolDef {
	out := make([]llm.ToolDef, len(r.defs))
	for i, fn := range r.defs {
		out[i] = llm.ToolDef{Type: "function", Function: fn}
	}
	return out
}

func toolDef(name, desc string, params json.RawMessage) json.RawMessage {
	fn := map[string]any{
		"name":        name,
		"description": desc,
	}
	if params != nil {
		fn["parameters"] = params
	}
	b, err := json.Marshal(fn)
	if err != nil {
		panic("marshal tool def: " + err.Error())
	}
	return b
}
