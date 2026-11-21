package httpx

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"
)

// Metrics provides dependency-free Prometheus text format instrumentation for Go microservices.
type Metrics struct {
	service string
	mu      sync.RWMutex
	reqs    map[metricKey]*metricVal
}

type metricKey struct {
	method string
	path   string
	status string
}

type metricVal struct {
	count      uint64
	durationMs float64
}

func NewMetrics(service string) *Metrics {
	return &Metrics{
		service: service,
		reqs:    make(map[metricKey]*metricVal),
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *statusWriter) Unwrap() http.ResponseWriter {
	return sw.ResponseWriter
}

func (sw *statusWriter) Flush() {
	if f, ok := sw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (m *Metrics) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" || r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		dur := float64(time.Since(start).Microseconds()) / 1000.0
		path := normalizeMetricPath(r.Pattern)
		if path == "" {
			path = r.URL.Path
		}
		k := metricKey{method: r.Method, path: path, status: strconv.Itoa(sw.status)}
		m.mu.Lock()
		v, ok := m.reqs[k]
		if !ok {
			v = &metricVal{}
			m.reqs[k] = v
		}
		v.count++
		v.durationMs += dur
		m.mu.Unlock()
	})
}

func normalizeMetricPath(pattern string) string {
	if pattern == "" {
		return ""
	}
	if _, after, ok := cutSpace(pattern); ok {
		return after
	}
	return pattern
}

func cutSpace(s string) (string, string, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			return s[:i], s[i+1:], true
		}
	}
	return "", s, false
}

func (m *Metrics) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		m.mu.RLock()
		keys := make([]metricKey, 0, len(m.reqs))
		for k := range m.reqs {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].path != keys[j].path {
				return keys[i].path < keys[j].path
			}
			if keys[i].method != keys[j].method {
				return keys[i].method < keys[j].method
			}
			return keys[i].status < keys[j].status
		})
		_, _ = fmt.Fprintf(w, "# HELP erp_http_requests_total Total HTTP requests processed.\n# TYPE erp_http_requests_total counter\n")
		for _, k := range keys {
			v := m.reqs[k]
			_, _ = fmt.Fprintf(w, "erp_http_requests_total{service=%q,method=%q,path=%q,status=%q} %d\n",
				m.service, k.method, k.path, k.status, v.count)
		}
		_, _ = fmt.Fprintf(w, "# HELP erp_http_request_duration_ms_sum Total HTTP request latency in ms.\n# TYPE erp_http_request_duration_ms_sum counter\n")
		for _, k := range keys {
			v := m.reqs[k]
			_, _ = fmt.Fprintf(w, "erp_http_request_duration_ms_sum{service=%q,method=%q,path=%q,status=%q} %.3f\n",
				m.service, k.method, k.path, k.status, v.durationMs)
		}
		m.mu.RUnlock()
	}
}

type ReadinessCheck func(context.Context) error

// ReadinessHandler executes dependency probes (e.g., DB Ping + Kafka Dial).
func ReadinessHandler(service string, checks ...ReadinessCheck) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		for _, chk := range checks {
			if chk == nil {
				continue
			}
			if err := chk(c); err != nil {
				WriteJSON(w, http.StatusServiceUnavailable, map[string]any{
					"status":  "not_ready",
					"service": service,
					"error":   err.Error(),
				})
				return
			}
		}
		WriteJSON(w, http.StatusOK, map[string]any{
			"status":  "ready",
			"service": service,
		})
	}
}

