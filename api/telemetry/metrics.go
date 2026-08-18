package telemetry

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var histogramBuckets = [...]float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// Default is the process-wide registry exported by the API endpoints and
// instrumentation points. It intentionally keeps only bounded label sets.
var Default = NewMetrics()

type Metrics struct {
	started time.Time
	mu      sync.Mutex
	request map[requestKey]*histogram
	deploy  map[string]*histogram
	ssh     map[string]*histogram
}

type requestKey struct {
	method string
	route  string
	status int
}

type histogram struct {
	Count   uint64
	Sum     float64
	Buckets [len(histogramBuckets)]uint64
}

func NewMetrics() *Metrics {
	return &Metrics{
		started: time.Now(),
		request: make(map[requestKey]*histogram),
		deploy:  make(map[string]*histogram),
		ssh:     make(map[string]*histogram),
	}
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		writer := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(writer, r)
		route := r.Pattern
		if route == "" {
			route = "unknown"
		}
		m.RecordHTTP(r.Method, route, writer.statusCode(), time.Since(started))
	})
}

func (m *Metrics) RecordHTTP(method, route string, status int, duration time.Duration) {
	if method == "" {
		method = "GET"
	}
	if route == "" {
		route = "unknown"
	}
	if status < 100 {
		status = http.StatusOK
	}
	m.mu.Lock()
	observe(m.requestFor(requestKey{method: method, route: route, status: status}), duration.Seconds())
	m.mu.Unlock()
}

func (m *Metrics) RecordDeployment(status string, duration time.Duration) {
	status = normalizeStatus(status, "failed")
	m.mu.Lock()
	observe(m.deploymentFor(status), duration.Seconds())
	m.mu.Unlock()
}

func (m *Metrics) RecordSSHConnection(success bool, duration time.Duration) {
	status := "failure"
	if success {
		status = "success"
	}
	m.mu.Lock()
	observe(m.sshFor(status), duration.Seconds())
	m.mu.Unlock()
}

func (m *Metrics) Handler(w http.ResponseWriter, _ *http.Request) {
	m.mu.Lock()
	request := cloneMap(m.request)
	deploy := cloneMap(m.deploy)
	ssh := cloneMap(m.ssh)
	uptime := time.Since(m.started).Seconds()
	m.mu.Unlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	var out strings.Builder
	fmt.Fprintln(&out, "# HELP uploy_up Whether the uploy API process is running.")
	fmt.Fprintln(&out, "# TYPE uploy_up gauge")
	fmt.Fprintln(&out, "uploy_up 1")
	fmt.Fprintln(&out, "# HELP uploy_process_uptime_seconds API process uptime in seconds.")
	fmt.Fprintln(&out, "# TYPE uploy_process_uptime_seconds gauge")
	fmt.Fprintf(&out, "uploy_process_uptime_seconds %s\n", formatFloat(uptime))

	fmt.Fprintln(&out, "# HELP uploy_http_requests_total Total HTTP requests handled by the API.")
	fmt.Fprintln(&out, "# TYPE uploy_http_requests_total counter")
	fmt.Fprintln(&out, "# HELP uploy_http_request_duration_seconds HTTP request duration in seconds.")
	fmt.Fprintln(&out, "# TYPE uploy_http_request_duration_seconds histogram")
	requestKeys := sortedKeys(request)
	for _, key := range requestKeys {
		labels := labelSet("method", key.method, "route", key.route, "status", strconv.Itoa(key.status))
		writeHistogram(&out, "uploy_http_requests_total", "uploy_http_request_duration_seconds", labels, request[key])
	}

	fmt.Fprintln(&out, "# HELP uploy_deployments_total Total deployments completed by status.")
	fmt.Fprintln(&out, "# TYPE uploy_deployments_total counter")
	fmt.Fprintln(&out, "# HELP uploy_deployment_duration_seconds Deployment duration in seconds.")
	fmt.Fprintln(&out, "# TYPE uploy_deployment_duration_seconds histogram")
	for _, status := range sortedStringKeys(deploy) {
		writeHistogram(&out, "uploy_deployments_total", "uploy_deployment_duration_seconds", labelSet("status", status), deploy[status])
	}

	fmt.Fprintln(&out, "# HELP uploy_ssh_connections_total SSH connection attempts by result.")
	fmt.Fprintln(&out, "# TYPE uploy_ssh_connections_total counter")
	fmt.Fprintln(&out, "# HELP uploy_ssh_connection_duration_seconds SSH connection duration in seconds.")
	fmt.Fprintln(&out, "# TYPE uploy_ssh_connection_duration_seconds histogram")
	for _, status := range sortedStringKeys(ssh) {
		writeHistogram(&out, "uploy_ssh_connections_total", "uploy_ssh_connection_duration_seconds", labelSet("status", status), ssh[status])
	}
	_, _ = w.Write([]byte(out.String()))
}

func (m *Metrics) requestFor(key requestKey) *histogram {
	if m.request[key] == nil {
		m.request[key] = &histogram{}
	}
	return m.request[key]
}

func (m *Metrics) deploymentFor(status string) *histogram {
	if m.deploy[status] == nil {
		m.deploy[status] = &histogram{}
	}
	return m.deploy[status]
}

func (m *Metrics) sshFor(status string) *histogram {
	if m.ssh[status] == nil {
		m.ssh[status] = &histogram{}
	}
	return m.ssh[status]
}

func observe(h *histogram, value float64) {
	if value < 0 {
		value = 0
	}
	h.Count++
	h.Sum += value
	for i, bucket := range histogramBuckets {
		if value <= bucket {
			h.Buckets[i]++
		}
	}
}

func writeHistogram(out *strings.Builder, counterName, histogramName, labels string, h *histogram) {
	fmt.Fprintf(out, "%s{%s} %d\n", counterName, labels, h.Count)
	for i, bucket := range histogramBuckets {
		fmt.Fprintf(out, "%s{%s,le=\"%s\"} %d\n", histogramName, labels, formatFloat(bucket), h.Buckets[i])
	}
	fmt.Fprintf(out, "%s{%s,le=\"+Inf\"} %d\n", histogramName, labels, h.Count)
	fmt.Fprintf(out, "%s_sum{%s} %s\n", histogramName, labels, formatFloat(h.Sum))
	fmt.Fprintf(out, "%s_count{%s} %d\n", histogramName, labels, h.Count)
}

func labelSet(values ...string) string {
	parts := make([]string, 0, len(values)/2)
	for i := 0; i+1 < len(values); i += 2 {
		parts = append(parts, values[i]+"=\""+escapeLabel(values[i+1])+"\"")
	}
	return strings.Join(parts, ",")
}

func escapeLabel(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return strings.ReplaceAll(value, "\n", `\n`)
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func normalizeStatus(value, fallback string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "success" || value == "failed" || value == "in_progress" {
		return value
	}
	return fallback
}

func cloneMap[T comparable](source map[T]*histogram) map[T]*histogram {
	clone := make(map[T]*histogram, len(source))
	for key, value := range source {
		copyValue := *value
		clone[key] = &copyValue
	}
	return clone
}

func sortedKeys(source map[requestKey]*histogram) []requestKey {
	keys := make([]requestKey, 0, len(source))
	for key := range source {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].method != keys[j].method {
			return keys[i].method < keys[j].method
		}
		if keys[i].route != keys[j].route {
			return keys[i].route < keys[j].route
		}
		return keys[i].status < keys[j].status
	})
	return keys
}

func sortedStringKeys(source map[string]*histogram) []string {
	keys := make([]string, 0, len(source))
	for key := range source {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

type responseWriter struct {
	http.ResponseWriter
	wroteHeader bool
	status      int
}

func (w *responseWriter) statusCode() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *responseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}

func (w *responseWriter) Flush() {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
