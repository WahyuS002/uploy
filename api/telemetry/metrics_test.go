package telemetry

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMetricsExposeInternalSignals(t *testing.T) {
	metrics := NewMetrics()
	metrics.RecordHTTP("GET", "/healthz", http.StatusOK, 25*time.Millisecond)
	metrics.RecordDeployment("success", 2*time.Second)
	metrics.RecordSSHConnection(false, 500*time.Millisecond)

	recorder := httptest.NewRecorder()
	metrics.Handler(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := recorder.Body.String()
	for _, want := range []string{
		"# TYPE uploy_http_requests_total counter",
		`uploy_http_requests_total{method="GET",route="/healthz",status="200"} 1`,
		`uploy_deployments_total{status="success"} 1`,
		`uploy_ssh_connections_total{status="failure"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics output missing %q", want)
		}
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/plain; version=0.0.4; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}
}

func TestMiddlewareRecordsStatusAndPreservesFlush(t *testing.T) {
	metrics := NewMetrics()
	handler := metrics.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("metrics middleware removed http.Flusher")
			return
		}
		w.WriteHeader(http.StatusAccepted)
		flusher.Flush()
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/custom", nil))
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusAccepted)
	}

	metricsRecorder := httptest.NewRecorder()
	metrics.Handler(metricsRecorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(metricsRecorder.Body.String(), `status="202"`) {
		t.Fatal("metrics output did not record accepted response")
	}
}
