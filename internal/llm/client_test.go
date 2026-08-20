package llm

import "testing"

func TestNew(t *testing.T) {
	t.Parallel()
	c := New("sk-test", "deepseek-test", false)
	if c == nil {
		t.Fatal("New returned nil")
	}
	if c.apiKey != "sk-test" {
		t.Errorf("apiKey = %q, want sk-test", c.apiKey)
	}
	if c.baseURL != "https://llmapi.paratera.com" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, defaultBaseURL)
	}
	// model is caller-supplied, no default to compare against
	_ = c.model
}
