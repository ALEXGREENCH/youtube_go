package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
)

// Registry is a minimal in-memory counter store.
type Registry struct {
	mu     sync.RWMutex
	counts map[string]uint64
}

// New creates an empty registry.
func New() *Registry {
	return &Registry{
		counts: make(map[string]uint64),
	}
}

// Wrap adds counting middleware around a handler.
func (r *Registry) Wrap(name string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.Inc(name)
		next.ServeHTTP(w, req)
	})
}

// Inc increments a named counter.
func (r *Registry) Inc(name string) {
	r.mu.Lock()
	r.counts[name]++
	r.mu.Unlock()
}

// Handler exposes counters as plain text.
func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		r.mu.RLock()
		keys := make([]string, 0, len(r.counts))
		for k := range r.counts {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(w, "%s %d\n", k, r.counts[k])
		}
		r.mu.RUnlock()
	})
}
