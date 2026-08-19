package monitoring

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// stubTransport points the agent plumbing at a local test server, standing in
// for the SSH tunnel a real call would ride.
type stubTransport struct {
	base string
}

func (s stubTransport) BaseURL() string { return s.base }

func (s stubTransport) Do(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req)
}

func newStub(t *testing.T, handler http.HandlerFunc) Transport {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return stubTransport{base: server.URL}
}

func TestOverSSHTargetsLoopback(t *testing.T) {
	transport := OverSSH(nil, 9500)
	if got := transport.BaseURL(); got != "http://127.0.0.1:9500" {
		t.Fatalf("BaseURL = %q; want the agent's loopback address on the server", got)
	}
}

func TestFetchDecodesAndAuthenticates(t *testing.T) {
	var gotAuth, gotPath, gotBody string
	transport := newStub(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		gotBody = string(buf)
		w.Write([]byte(`{"points":[{"deployment_id":"dep-1","cpu_percent":12.5}]}`))
	})

	var response HistoryResponse
	err := fetch(context.Background(), transport, agentCall{
		method:  http.MethodPost,
		path:    "/v1/history",
		token:   "control-token",
		payload: []byte(`{"deployment_ids":["dep-1"]}`),
		timeout: time.Second,
		limit:   historyLimit,
	}, &response)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}

	if gotAuth != "Bearer control-token" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotPath != "/v1/history" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotBody != `{"deployment_ids":["dep-1"]}` {
		t.Fatalf("body = %q", gotBody)
	}
	if len(response.Points) != 1 || response.Points[0].CPUPercent != 12.5 {
		t.Fatalf("points = %+v", response.Points)
	}
}

// The agent's own message is the useful half of a failure — "retention must be
// between 1 and 30" tells an operator what to do, "HTTP 400" does not.
func TestFetchSurfacesAgentMessage(t *testing.T) {
	transport := newStub(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("max_points is out of range"))
	})

	err := fetch(context.Background(), transport, agentCall{
		method: http.MethodGet, path: "/v1/latest", token: "t", timeout: time.Second, limit: latestLimit,
	}, &LatestResponse{})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "max_points is out of range") {
		t.Fatalf("err = %v; want the agent's own message", err)
	}
	if !strings.Contains(err.Error(), "400") {
		t.Fatalf("err = %v; want the status code kept", err)
	}
}

func TestFetchExpectsNoContentWhenDestIsNil(t *testing.T) {
	transport := newStub(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	if err := fetch(context.Background(), transport, agentCall{
		method: http.MethodDelete, path: "/v1/history", token: "t", timeout: time.Second,
	}, nil); err != nil {
		t.Fatalf("fetch: %v", err)
	}

	rejected := newStub(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	if err := fetch(context.Background(), rejected, agentCall{
		method: http.MethodDelete, path: "/v1/history", token: "t", timeout: time.Second,
	}, nil); err == nil {
		t.Fatal("expected 200 to be rejected where 204 is required")
	}
}

// An agent that accepts the connection and then stalls must not hold a dashboard
// request open indefinitely.
func TestFetchAppliesCallTimeout(t *testing.T) {
	release := make(chan struct{})
	transport := newStub(t, func(w http.ResponseWriter, r *http.Request) {
		<-release
	})
	// Registered after newStub so it runs before the server is closed: cleanups
	// run last-in-first-out, and closing a server with a stalled handler blocks.
	t.Cleanup(func() { close(release) })

	started := time.Now()
	err := fetch(context.Background(), transport, agentCall{
		method: http.MethodGet, path: "/v1/latest", token: "t", timeout: 100 * time.Millisecond, limit: latestLimit,
	}, &LatestResponse{})
	if err == nil {
		t.Fatal("expected a timeout")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("waited %v; the call timeout was not applied", elapsed)
	}
}
