// Package enginemcp gives the web BFF a narrow client for the engine MCP
// service. Browser code never receives the downstream MCP token.
package enginemcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const protocolVersion = "2026-07-28"

var ErrUnavailable = errors.New("engine MCP unavailable")

type ToolError struct {
	Message string
}

func (e *ToolError) Error() string {
	return e.Message
}

type Client struct {
	endpoint   string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		endpoint:   strings.TrimRight(baseURL, "/") + "/mcp",
		token:      token,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *Client) CallTool(ctx context.Context, name string, arguments json.RawMessage) (json.RawMessage, error) {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      name,
			"arguments": arguments,
			"_meta": map[string]any{
				"io.modelcontextprotocol/protocolVersion":    protocolVersion,
				"io.modelcontextprotocol/clientInfo":         map[string]string{"name": "liki-web", "version": "1"},
				"io.modelcontextprotocol/clientCapabilities": map[string]any{},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode engine MCP request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build engine MCP request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("MCP-Protocol-Version", protocolVersion)
	request.Header.Set("Mcp-Method", "tools/call")
	request.Header.Set("Mcp-Name", name)
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return nil, ErrUnavailable
	}

	raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, ErrUnavailable
	}
	envelope, err := decodeEnvelope(raw, response.Header.Get("Content-Type"))
	if err != nil {
		return nil, ErrUnavailable
	}
	if envelope.Error != nil {
		return nil, &ToolError{Message: envelope.Error.Message}
	}
	if envelope.Result.IsError {
		return nil, &ToolError{Message: firstText(envelope.Result.Content)}
	}
	result := firstText(envelope.Result.Content)
	if result == "" {
		return nil, ErrUnavailable
	}
	var data json.RawMessage
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		return nil, ErrUnavailable
	}
	return data, nil
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type callResult struct {
	Content []content `json:"content"`
	IsError bool      `json:"isError"`
}

type envelope struct {
	Result callResult `json:"result"`
	Error  *rpcError  `json:"error"`
}

func decodeEnvelope(raw []byte, contentType string) (envelope, error) {
	var output envelope
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "text/event-stream") {
		scanner := bufio.NewScanner(bytes.NewReader(raw))
		var found bool
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			value := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if value == "" {
				continue
			}
			if err := json.Unmarshal([]byte(value), &output); err != nil {
				return output, err
			}
			found = true
		}
		if err := scanner.Err(); err != nil {
			return output, err
		}
		if !found {
			return output, io.ErrUnexpectedEOF
		}
		return output, nil
	}
	if err := json.Unmarshal(raw, &output); err != nil {
		return output, err
	}
	return output, nil
}

func firstText(items []content) string {
	for _, item := range items {
		if item.Type == "text" {
			return item.Text
		}
	}
	return ""
}
