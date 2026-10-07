package discovery

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"
)

// Limiter has a bounded source table and no timer goroutines. Expired entries
// are evicted on demand; a full table fails closed instead of allocating more.
type Limiter struct {
	mu       sync.Mutex
	entries  map[string]time.Time
	Interval time.Duration
}

func (l *Limiter) Allow(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.entries == nil {
		l.entries = map[string]time.Time{}
	}
	if until := l.entries[host]; now.Before(until) {
		return false
	}
	if len(l.entries) >= 1024 {
		for k, v := range l.entries {
			if !now.Before(v) {
				delete(l.entries, k)
			}
		}
		if len(l.entries) >= 1024 {
			return false
		}
	}
	l.entries[host] = now.Add(l.Interval)
	return true
}
func Handler(info func() Info) http.Handler {
	l := &Limiter{Interval: time.Second}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.Header().Set("Allow", "GET")
			http.Error(w, "method not allowed", 405)
			return
		}
		if r.ContentLength != 0 || len(r.URL.RawQuery) > 0 {
			http.Error(w, "unexpected body or query", 400)
			return
		}
		if !l.Allow(r.RemoteAddr) {
			http.Error(w, "rate limited", 429)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(info())
	})
}
