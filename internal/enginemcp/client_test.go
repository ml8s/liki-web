package enginemcp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ml8s/liki-web/internal/enginemcp"
)

func TestCallToolSendsMCPHeadersAndUnwrapsText(t *testing.T) {
	var gotPath, gotMethod, gotName, gotAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Header.Get("Mcp-Method")
		gotName = r.Header.Get("Mcp-Name")
		gotAuthorization = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"{\"ok\":true}"}]}}`))
	}))
	defer server.Close()

	client := enginemcp.NewClient(server.URL, "test-token")
	data, err := client.CallTool(context.Background(), "bazi_chart", []byte(`{"solar_time":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"ok":true}` {
		t.Fatalf("data = %s", data)
	}
	if gotPath != "/mcp" || gotMethod != "tools/call" || gotName != "bazi_chart" {
		t.Fatalf("MCP request = path %q method %q name %q", gotPath, gotMethod, gotName)
	}
	if gotAuthorization != "Bearer test-token" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
}

func TestCallToolSupportsEventStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: message\n"))
		_, _ = w.Write([]byte("data: {\"jsonrpc\":\"2.0\",\"result\":{\"content\":[{\"type\":\"text\",\"text\":\"[1,2]\"}]}}\n\n"))
	}))
	defer server.Close()

	data, err := enginemcp.NewClient(server.URL, "").CallTool(context.Background(), "huangli_days", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "[1,2]" {
		t.Fatalf("data = %s", data)
	}
}
