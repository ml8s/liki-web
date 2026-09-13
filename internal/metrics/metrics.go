// Package metrics provides a tiny, dependency-free observability surface for
// payment, fulfillment, and notification lifecycle counters.
package metrics

import (
	"encoding/json"
	"net/http"
	"sort"
	"sync"
)

type Registry struct {
	mu       sync.RWMutex
	counters map[string]uint64
}

var Default = New()

func New() *Registry { return &Registry{counters: map[string]uint64{}} }

func (r *Registry) Inc(name string, labels map[string]string) {
	key := name + labelsKey(labels)
	r.mu.Lock()
	r.counters[key]++
	r.mu.Unlock()
}

func (r *Registry) Snapshot() map[string]uint64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]uint64, len(r.counters))
	for k, v := range r.counters {
		out[k] = v
	}
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
	out := ""
	for _, k := range keys {
		out += "|" + k + "=" + labels[k]
	}
	return out
}

// Handler rejects all requests when no token is configured. Production should
// expose this through an internal-only network path or an authenticated proxy.
func Handler(token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+token && r.URL.Query().Get("token") != token {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"counters": Default.Snapshot()})
	}
}
