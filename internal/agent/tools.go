// tools.go — LLM Chat Tool Registry for liki-web
// Tools are discovered lazily from the engine via rpc.discover on first use.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"liki-web/internal/llm"
)

const rpcDiscover = "rpc.discover"

// ChatToolRegistry executes LLM tool calls by forwarding them as JSON-RPC
// requests to the liki engine server. Tools are discovered lazily via
// rpc.discover on first use.
type ChatToolRegistry struct {
	engineURL     string
	client        *http.Client
	handlers      map[string]func(context.Context, json.RawMessage) (json.RawMessage, error)
	cachedSchemas []llm.ToolDef

	initOnce sync.Once
}

// NewToolRegistry creates a tool registry. Tools are not discovered until
// Schemas() or Execute() is called for the first time. If the engine is
// unavailable at discovery time, tools remain empty and subsequent calls
// use whatever was cached (possibly empty).
func NewToolRegistry(engineURL string) *ChatToolRegistry {
	if engineURL == "" {
		engineURL = os.Getenv("LIKI_ENGINE_URL")
	}
	if engineURL == "" {
		engineURL = "http://localhost:8080/jsonrpc"
	}

	return &ChatToolRegistry{
		engineURL: engineURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		handlers: map[string]func(context.Context, json.RawMessage) (json.RawMessage, error){},
	}
}

// Register adds a built-in tool that does not come from engine discovery.
// Must be called before Schemas() or Execute() triggers discoverEngineTools.
func (r *ChatToolRegistry) Register(name, desc string, params json.RawMessage, handler func(context.Context, json.RawMessage) (json.RawMessage, error)) {
	r.handlers[name] = handler
	r.addSchema(toolDef(name, desc, params))
}

func (r *ChatToolRegistry) addSchema(def json.RawMessage) {
	if def == nil {
		return
	}
	r.cachedSchemas = append(r.cachedSchemas, llm.ToolDef{Type: llm.ToolTypeFunction, Function: def})
}

// ReadFileHandler returns a tool handler that reads files from the given
// directory. It rejects paths that escape the directory (.., absolute paths).
func ReadFileHandler(root string) func(context.Context, json.RawMessage) (json.RawMessage, error) {
	return func(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
		var p struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return nil, fmt.Errorf("read_file: invalid args: %w", err)
		}
		if p.Path == "" {
			return nil, fmt.Errorf("read_file: path is required")
		}
		fullPath := filepath.Join(root, p.Path)
		if !filepath.IsLocal(p.Path) || !strings.HasPrefix(fullPath, root) {
			return nil, fmt.Errorf("read_file: path escapes skills directory")
		}
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("read_file: %w", err)
		}
		// 命理向契约：直接返回文件内容本体（JSON 字符串），不带 {content} 壳。
		data, err := json.Marshal(string(content))
		if err != nil {
			return nil, fmt.Errorf("read_file: marshal: %w", err)
		}
		return json.RawMessage(data), nil
	}
}

// discoverEngineTools discovers engine tools on first call. Safe for concurrent use.
func (r *ChatToolRegistry) discoverEngineTools() {
	r.initOnce.Do(func() {
		discovered, err := r.discoverTools()
		if err != nil {
			slog.Warn("liki-web: tool discovery failed, engine tools unavailable", "err", err)
			return
		}

		for rpcMethod, info := range discovered {
			if rpcMethod == rpcDiscover {
				continue
			}
			toolName := strings.ReplaceAll(rpcMethod, ".", "_")
			method := rpcMethod
			r.handlers[toolName] = func(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
				result, err := r.callEngine(ctx, method, args)
				if err != nil {
					return nil, err
				}
				// 命理向契约：引擎返回 {"_product":..., "data":...}，这里只把
				// 命理本体 data 交给 LLM，去掉传输壳。
				return unwrapEngineResult(result)
			}
			r.addSchema(toolDef(toolName, info.Description, info.Params))
		}
	})
}

// unwrapEngineResult extracts the 命理本体 from the engine's wrapped result
// {"_product":..., "data":...}. If the shape is unexpected, the raw result is
// passed through rather than failing a compute call.
func unwrapEngineResult(raw json.RawMessage) (json.RawMessage, error) {
	var wrapped struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil && wrapped.Data != nil && len(wrapped.Data) > 0 {
		return wrapped.Data, nil
	}
	return raw, nil
}

type discoveredMethod struct {
	Description string
	Params      json.RawMessage
}

// discoverTools calls rpc.discover and returns a map of RPC method name → info.
// Retries up to 3 times with 2s backoff. Returns error if engine is unreachable.
func (r *ChatToolRegistry) discoverTools() (map[string]discoveredMethod, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var lastErr error
	for i := 0; i < 3; i++ {
		result, err := r.callEngine(ctx, rpcDiscover, nil)
		if err != nil {
			lastErr = err
			if i < 2 {
				timer := time.NewTimer(2 * time.Second)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil, fmt.Errorf("rpc.discover: %w", ctx.Err())
				case <-timer.C:
				}
			}
			continue
		}

		var doc struct {
			Methods []struct {
				Name        string          `json:"name"`
				Description string          `json:"description"`
				Params      json.RawMessage `json:"params"`
			} `json:"methods"`
		}
		if err := json.Unmarshal(result, &doc); err != nil {
			return nil, fmt.Errorf("parse rpc.discover: %w", err)
		}

		m := make(map[string]discoveredMethod, len(doc.Methods))
		for _, method := range doc.Methods {
			m[method.Name] = discoveredMethod{
				Description: method.Description,
				Params:      method.Params,
			}
		}
		return m, nil
	}

	return nil, fmt.Errorf("rpc.discover failed after 3 retries: %w", lastErr)
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
	defer func() { _ = resp.Body.Close() }()

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
	r.discoverEngineTools()
	h, ok := r.handlers[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
	return h(ctx, args)
}

func (r *ChatToolRegistry) Schemas() []llm.ToolDef {
	r.discoverEngineTools()
	return r.cachedSchemas
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
		// json.Marshal on map[string]any with RawMessage values never fails in
		// practice; log + return nil to avoid crashing the process.
		slog.Error("marshal tool def", "tool", name, "err", err)
		return nil
	}
	return b
}
