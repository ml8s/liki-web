// Package metrics provides a tiny, dependency-free observability surface for
// payment, fulfillment, and notification lifecycle counters.
package metrics

import (
	"crypto/subtle"
	"fmt"
	"maps"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// Metric names are owned by the cross-service Liki observability contract.
const (
	MetricHTTPRequests              = "liki_web_http_requests_total"
	MetricHTTPRequestDuration       = "liki_web_http_request_duration_seconds_total"
	MetricChatAgentRuns             = "liki_web_chat_agent_runs_total"
	MetricChatAgentLLMCalls         = "liki_web_chat_agent_llm_calls_total"
	MetricChatAgentPromptTokens     = "liki_web_chat_agent_prompt_tokens_total"
	MetricChatAgentCompletionTokens = "liki_web_chat_agent_completion_tokens_total"
	MetricChatAgentThoughtTokens    = "liki_web_chat_agent_thought_tokens_total"
	MetricAnalyticsPageview         = "liki_web_analytics_pageview_total"
	MetricAppFlowEvent              = "liki_web_app_flow_event_total"
	MetricClientError               = "liki_web_client_error_total"
)

var Default = New()

type Registry struct {
	mu        sync.RWMutex
	counters  map[string]uint64
	durations map[string]float64
}

func New() *Registry {
	return &Registry{counters: map[string]uint64{}, durations: map[string]float64{}}
}

func (r *Registry) Inc(name string, labels map[string]string) {
	r.Add(name, labels, 1)
}

// Add 将计数器增加 delta（耗时累计等非 1 增量场景）。
func (r *Registry) Add(name string, labels map[string]string, delta uint64) {
	key := name + labelsKey(labels)
	r.mu.Lock()
	r.counters[key] += delta
	r.mu.Unlock()
}

func (r *Registry) Snapshot() map[string]uint64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]uint64, len(r.counters))
	maps.Copy(out, r.counters)
	return out
}

// ObserveDurationSeconds accumulates a duration counter in seconds, as required
// by the Prometheus-compatible Liki metric contract.
func (r *Registry) ObserveDurationSeconds(name string, labels map[string]string, seconds float64) {
	key := name + labelsKey(labels)
	r.mu.Lock()
	r.durations[key] += seconds
	r.mu.Unlock()
}

func (r *Registry) DurationSnapshot() map[string]float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]float64, len(r.durations))
	maps.Copy(out, r.durations)
	return out
}

func labelsKey(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out strings.Builder
	for _, k := range keys {
		out.WriteString("|" + k + "=" + labels[k])
	}
	return out.String()
}

// RenderPrometheus converts the internal snapshot into the standard text
// exposition format. Keeping this dependency-free avoids pulling a registry
// abstraction through every domain package.
func RenderPrometheus(counters map[string]uint64, durations map[string]float64) string {
	type family struct {
		samples []string
	}
	families := make(map[string]*family)
	names := make([]string, 0)

	add := func(key, value string) {
		metric, rawLabels, _ := strings.Cut(key, "|")
		labels := make([]string, 0)
		if rawLabels != "" {
			for pair := range strings.SplitSeq(rawLabels, "|") {
				name, labelValue, ok := strings.Cut(pair, "=")
				if !ok {
					continue
				}
				labels = append(labels, fmt.Sprintf(
					`%s=%q`, escapePrometheusLabelName(name), escapePrometheusLabelValue(labelValue),
				))
			}
		}
		sample := metric
		if len(labels) > 0 {
			sample += "{" + strings.Join(labels, ",") + "}"
		}
		sample += " " + value + "\n"
		item, ok := families[metric]
		if !ok {
			item = &family{}
			families[metric] = item
			names = append(names, metric)
		}
		item.samples = append(item.samples, sample)
	}

	for key, value := range counters {
		add(key, fmt.Sprintf("%d", value))
	}
	durationKeys := make([]string, 0, len(durations))
	for key := range durations {
		durationKeys = append(durationKeys, key)
	}
	sort.Strings(durationKeys)
	for _, key := range durationKeys {
		add(key, fmt.Sprintf("%f", durations[key]))
	}
	sort.Strings(names)
	out := strings.Builder{}
	for _, name := range names {
		out.WriteString("# TYPE ")
		out.WriteString(name)
		out.WriteString(" counter\n")
		item := families[name]
		sort.Strings(item.samples)
		for _, sample := range item.samples {
			out.WriteString(sample)
		}
	}
	return out.String()
}

func escapePrometheusLabelName(value string) string {
	var out strings.Builder
	for _, r := range value {
		if r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			out.WriteRune(r)
		} else {
			out.WriteRune('_')
		}
	}
	return out.String()
}

func escapePrometheusLabelValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

// Handler rejects all requests when no token is configured. Authentication uses
// a bearer token only, keeping the secret out of URLs and access logs. Successful
// output is always Prometheus text exposition version 0.0.4.
func Handler(token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			http.NotFound(w, r)
			return
		}
		expected := "Bearer " + token
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte(expected)) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = w.Write([]byte(RenderPrometheus(Default.Snapshot(), Default.DurationSnapshot())))
	}
}
