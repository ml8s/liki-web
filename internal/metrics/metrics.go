// Package metrics provides a tiny, dependency-free observability surface for
// payment, fulfillment, and notification lifecycle counters.
package metrics

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/ml8s/liki-web/internal/httputil"
)

type Registry struct {
	mu       sync.RWMutex
	counters map[string]uint64
}

var Default = New()

func New() *Registry { return &Registry{counters: map[string]uint64{}} }

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
func RenderPrometheus(counters map[string]uint64) string {
	type family struct {
		samples []string
	}
	families := make(map[string]*family)
	names := make([]string, 0)
	for key, value := range counters {
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
		sample += fmt.Sprintf(" %d\n", value)
		item, ok := families[metric]
		if !ok {
			item = &family{}
			families[metric] = item
			names = append(names, metric)
		}
		item.samples = append(item.samples, sample)
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
// a bearer token only, keeping the secret out of URLs and access logs.
func Handler(token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			http.NotFound(w, r)
			return
		}
		expected := "Bearer " + token
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte(expected)) != 1 {
			httputil.RespondError(w, http.StatusForbidden, "forbidden", "forbidden")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if r.URL.Query().Get("format") == "prometheus" || strings.Contains(r.Header.Get("Accept"), "text/plain") {
			w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
			_, _ = w.Write([]byte(RenderPrometheus(Default.Snapshot())))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"counters": Default.Snapshot()})
	}
}
