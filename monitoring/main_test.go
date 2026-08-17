package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseBytes(t *testing.T) {
	tests := map[string]int64{"1.5kB": 1500, "2MiB": 2 << 20, "3GB": 3_000_000_000}
	for input, want := range tests {
		got, err := parseBytes(input)
		if err != nil || got != want {
			t.Fatalf("parseBytes(%q) = %d, %v; want %d", input, got, err, want)
		}
	}
}

func TestIsSQLiteFull(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	if _, err := database.Exec("PRAGMA max_page_count = 2"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("CREATE TABLE full_test (payload BLOB)"); err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec("INSERT INTO full_test VALUES (zeroblob(1048576))")
	if err == nil {
		t.Fatal("expected SQLITE_FULL")
	}
	if !isSQLiteFull(err) {
		t.Fatalf("isSQLiteFull(%T: %v) = false", err, err)
	}
}

func TestRequireTokenAcceptsBothScopes(t *testing.T) {
	handler := requireToken(config{controlToken: "control", readerToken: "reader"}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	for _, token := range []string{"control", "reader"} {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != http.StatusNoContent {
			t.Fatalf("token %q status = %d", token, response.Code)
		}
	}
}

func TestReaderCannotDeleteHistory(t *testing.T) {
	handler := requireControlToken(config{controlToken: "control", readerToken: "reader"}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodDelete, "/v1/history", nil)
	req.Header.Set("Authorization", "Bearer reader")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("reader delete status = %d", response.Code)
	}
}

func TestLoadConfigRejectsSharedTokens(t *testing.T) {
	t.Setenv("UPLOY_MONITOR_CONTROL_TOKEN", strings.Repeat("a", 32))
	t.Setenv("UPLOY_MONITOR_READER_TOKEN", strings.Repeat("a", 32))
	if _, err := loadConfig(); err == nil {
		t.Fatal("expected shared tokens to be rejected")
	}
}

func TestDownsampleCapsPoints(t *testing.T) {
	from := time.Now().Add(-time.Hour).UnixMilli()
	raw := make([]sample, 120)
	for index := range raw {
		raw[index] = sample{SampledAt: from + int64(index)*30_000}
	}
	got := downsample(raw, from, from+int64(time.Hour/time.Millisecond), 10)
	if len(got) > 10 {
		t.Fatalf("downsample returned %d points, want at most 10", len(got))
	}
}

func TestDownsampleCapsInclusiveRange(t *testing.T) {
	raw := make([]sample, 11)
	for index := range raw {
		raw[index] = sample{SampledAt: int64(index)}
	}
	if got := downsample(raw, 0, 10, 10); len(got) > 10 {
		t.Fatalf("downsample returned %d points, want at most 10", len(got))
	}
}

func TestHistoriesHandlerGroupsDeployments(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.Exec(`CREATE TABLE deployment_metrics (
		deployment_id TEXT, sampled_at INTEGER, container_id TEXT, container_name TEXT, state TEXT,
		cpu_percent REAL, memory_used_bytes INTEGER, memory_limit_bytes INTEGER,
		network_in_bytes_total INTEGER, network_out_bytes_total INTEGER, uptime_seconds INTEGER
	)`); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().UnixMilli()
	for _, deploymentID := range []string{"deployment-a", "deployment-b"} {
		if _, err := database.Exec(`INSERT INTO deployment_metrics VALUES (?, ?, 'container', 'app', 'running', 1, 2, 3, 4, 5, 6)`, deploymentID, now); err != nil {
			t.Fatal(err)
		}
	}
	collector := &collector{db: database}
	query, err := json.Marshal(historyQuery{
		DeploymentIDs: []string{"deployment-a", "deployment-b"},
		From:          time.UnixMilli(now - 1000).UTC().Format(time.RFC3339),
		To:            time.UnixMilli(now + 1000).UTC().Format(time.RFC3339),
		MaxPoints:     100,
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/history", strings.NewReader(string(query)))
	response := httptest.NewRecorder()
	collector.historiesQueryHandler(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body.String())
	}
	var body struct {
		Deployments map[string]historyResponse `json:"deployments"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Deployments["deployment-a"].Points) != 1 || len(body.Deployments["deployment-b"].Points) != 1 {
		t.Fatalf("unexpected histories: %+v", body.Deployments)
	}
}

func TestHistoriesHandlerCapsBatchPoints(t *testing.T) {
	deploymentIDs := make([]string, 11)
	for index := range deploymentIDs {
		deploymentIDs[index] = fmt.Sprintf("deployment-%d", index)
	}
	body, err := json.Marshal(historyQuery{DeploymentIDs: deploymentIDs, MaxPoints: maxHistoryPoints})
	if err != nil {
		t.Fatal(err)
	}
	collector := &collector{}
	request := httptest.NewRequest(http.MethodPost, "/v1/history", strings.NewReader(string(body)))
	response := httptest.NewRecorder()
	collector.historiesQueryHandler(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body.String())
	}
}
