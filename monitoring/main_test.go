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

func TestServerLatestHandler(t *testing.T) {
	collector := &collector{server: serverSample{
		SampledAt: 123, DiskUsedBytes: 8, DiskTotalBytes: 10, DiskUsedPercent: 87.5,
		Load1: 2.4, SwapUsedBytes: 512, SwapTotalBytes: 2048,
		Partitions: []serverPartition{{Mountpoint: "/", UsedPercent: 87.5}},
	}}
	response := httptest.NewRecorder()
	collector.serverLatestHandler(response, httptest.NewRequest(http.MethodGet, "/v1/server/latest", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	var got serverSample
	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.DiskUsedPercent != 87.5 {
		t.Fatalf("disk_used_percent = %v, want 87.5", got.DiskUsedPercent)
	}
	if got.DiskUsedBytes != 8 || got.Load1 != 2.4 || len(got.Partitions) != 1 {
		t.Fatalf("server metrics = %+v", got)
	}
}

func TestTrackFilesystemFiltersVirtualFilesystems(t *testing.T) {
	for _, filesystem := range []string{"proc", "sysfs", "tmpfs", "overlay", "cgroup2"} {
		if trackFilesystem(filesystem) {
			t.Fatalf("trackFilesystem(%q) = true", filesystem)
		}
	}
	for _, filesystem := range []string{"ext4", "xfs", "btrfs"} {
		if !trackFilesystem(filesystem) {
			t.Fatalf("trackFilesystem(%q) = false", filesystem)
		}
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

func TestCompactAndExpireRollsUpExistingMetrics(t *testing.T) {
	database, err := openDatabase(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	now := time.Date(2026, time.August, 18, 18, 20, 0, 0, time.UTC)
	start := time.Date(2026, time.August, 18, 8, 0, 0, 0, time.UTC)
	for index := 0; index < 10; index++ {
		metric := sample{
			DeploymentID:         "deployment-a",
			ContainerID:          "container-a",
			ContainerName:        "app",
			State:                "running",
			SampledAt:            start.Add(time.Duration(index) * time.Minute).UnixMilli(),
			CPUPercent:           float64(index),
			MemoryUsedBytes:      int64(100 + index),
			MemoryLimitBytes:     200,
			NetworkInBytesTotal:  int64(1_000 + index),
			NetworkOutBytesTotal: int64(2_000 + index),
			UptimeSeconds:        int64(3_000 + index),
		}
		insertMetric(t, database, "deployment_metrics", 0, metric)
	}

	collector := &collector{db: database, retention: 30 * 24 * time.Hour}
	if err := collector.compactAndExpire(now); err != nil {
		t.Fatal(err)
	}

	var cpu float64
	var networkIn int64
	if err := database.QueryRow(`SELECT cpu_percent, network_in_bytes_total
		FROM deployment_metric_rollups
		WHERE deployment_id = ? AND resolution_minutes = 480 AND sampled_at = ?`, "deployment-a", start.UnixMilli()).Scan(&cpu, &networkIn); err != nil {
		t.Fatal(err)
	}
	if cpu != 4.5 {
		t.Fatalf("480m cpu = %v, want 4.5", cpu)
	}
	if networkIn != 1_009 {
		t.Fatalf("480m network_in = %d, want 1009", networkIn)
	}

	var rawCount int
	if err := database.QueryRow("SELECT COUNT(*) FROM deployment_metrics").Scan(&rawCount); err != nil {
		t.Fatal(err)
	}
	if rawCount != 0 {
		t.Fatalf("raw metrics after compaction = %d, want 0", rawCount)
	}

	points, err := collector.queryHistoryAt("deployment-a", now.Add(-12*time.Hour).UnixMilli(), now.UnixMilli(), 100, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 1 || points[0].CPUPercent != 4.5 {
		t.Fatalf("history after compaction = %+v, want the 480m rollup", points)
	}
}

func TestQueryHistoryUsesEachRetentionTier(t *testing.T) {
	database, err := openDatabase(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	now := time.Date(2026, time.August, 18, 18, 20, 0, 0, time.UTC)
	collector := &collector{db: database, retention: 30 * 24 * time.Hour}
	insertMetric(t, database, "deployment_metrics", 0, testMetric(now.Add(-30*time.Minute), 1))
	insertMetric(t, database, "deployment_metric_rollups", 10, testMetric(now.Add(-3*time.Hour), 2))
	insertMetric(t, database, "deployment_metric_rollups", 20, testMetric(now.Add(-18*time.Hour), 3))
	insertMetric(t, database, "deployment_metric_rollups", 120, testMetric(now.Add(-2*24*time.Hour), 4))
	insertMetric(t, database, "deployment_metric_rollups", 480, testMetric(now.Add(-10*24*time.Hour), 5))

	points, err := collector.queryHistoryAt("deployment-a", now.Add(-20*24*time.Hour).UnixMilli(), now.UnixMilli(), 100, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 5 {
		t.Fatalf("history points = %d, want 5: %+v", len(points), points)
	}
	for index, want := range []float64{5, 4, 3, 2, 1} {
		if points[index].CPUPercent != want {
			t.Fatalf("point %d cpu = %v, want %v", index, points[index].CPUPercent, want)
		}
	}
}

func TestQueryHistoryFallsBackToExistingRawData(t *testing.T) {
	database, err := openDatabase(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	now := time.Date(2026, time.August, 18, 18, 20, 0, 0, time.UTC)
	collector := &collector{db: database, retention: 30 * 24 * time.Hour}
	insertMetric(t, database, "deployment_metrics", 0, testMetric(now.Add(-10*24*time.Hour), 7))

	points, err := collector.queryHistoryAt("deployment-a", now.Add(-11*24*time.Hour).UnixMilli(), now.UnixMilli(), 100, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 1 || points[0].CPUPercent != 7 {
		t.Fatalf("raw fallback history = %+v, want the existing raw point", points)
	}
}

func TestCompactAndExpireHonorsRetentionLimit(t *testing.T) {
	database, err := openDatabase(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	now := time.Date(2026, time.August, 18, 18, 20, 0, 0, time.UTC)
	insertMetric(t, database, "deployment_metric_rollups", 480, testMetric(now.Add(-8*24*time.Hour), 1))
	insertMetric(t, database, "deployment_metric_rollups", 480, testMetric(now.Add(-6*24*time.Hour), 2))
	collector := &collector{db: database, retention: 7 * 24 * time.Hour}
	if err := collector.compactAndExpire(now); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM deployment_metric_rollups WHERE resolution_minutes = 480").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("480m rollups after 7d retention = %d, want 1", count)
	}
}

func TestDeleteHistoryHandlerRemovesRollups(t *testing.T) {
	database, err := openDatabase(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	now := time.Now().UTC()
	insertMetric(t, database, "deployment_metrics", 0, testMetric(now, 1))
	insertMetric(t, database, "deployment_metric_rollups", 10, testMetric(now.Add(-time.Hour), 2))
	insertServerMetric(t, database, "server_metrics", 0, testServerMetric(now, 1))
	insertServerMetric(t, database, "server_metric_rollups", 10, testServerMetric(now.Add(-time.Hour), 2))
	collector := &collector{db: database}
	response := httptest.NewRecorder()
	collector.deleteHistoryHandler(response, httptest.NewRequest(http.MethodDelete, "/v1/history", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body.String())
	}
	for _, table := range []string{"deployment_metrics", "deployment_metric_rollups", "server_metrics", "server_metric_rollups"} {
		var count int
		if err := database.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s count = %d, want 0", table, count)
		}
	}
}

func TestCompactAndQueryServerHistory(t *testing.T) {
	database, err := openDatabase(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	now := time.Date(2026, time.August, 18, 18, 20, 0, 0, time.UTC)
	start := time.Date(2026, time.August, 18, 8, 0, 0, 0, time.UTC)
	for index := 0; index < 10; index++ {
		insertServerMetric(t, database, "server_metrics", 0, testServerMetric(start.Add(time.Duration(index)*time.Minute), float64(index)))
	}
	collector := &collector{db: database, retention: 30 * 24 * time.Hour}
	if err := collector.compactAndExpire(now); err != nil {
		t.Fatal(err)
	}

	var loadAverage float64
	if err := database.QueryRow(`SELECT load_1 FROM server_metric_rollups WHERE resolution_minutes = 480 AND sampled_at = ?`, start.UnixMilli()).Scan(&loadAverage); err != nil {
		t.Fatal(err)
	}
	if loadAverage != 4.5 {
		t.Fatalf("server 480m load = %v, want 4.5", loadAverage)
	}
	points, err := collector.queryServerHistoryAt(now.Add(-12*time.Hour).UnixMilli(), now.UnixMilli(), 100, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 1 || points[0].Load1 != 4.5 || len(points[0].Partitions) != 1 {
		t.Fatalf("server history = %+v", points)
	}
}

func testMetric(sampledAt time.Time, cpu float64) sample {
	return sample{
		DeploymentID:         "deployment-a",
		ContainerID:          "container-a",
		ContainerName:        "app",
		State:                "running",
		SampledAt:            sampledAt.UnixMilli(),
		CPUPercent:           cpu,
		MemoryUsedBytes:      100,
		MemoryLimitBytes:     200,
		NetworkInBytesTotal:  300,
		NetworkOutBytesTotal: 400,
		UptimeSeconds:        500,
	}
}

func testServerMetric(sampledAt time.Time, loadAverage float64) serverSample {
	return serverSample{
		SampledAt: sampledAt.UnixMilli(), DiskUsedBytes: 100, DiskTotalBytes: 200, DiskUsedPercent: 50,
		DiskReadBytesTotal: 300, DiskWriteBytesTotal: 400, Load1: loadAverage, Load5: loadAverage + 1,
		Load15: loadAverage + 2, SwapUsedBytes: 500, SwapTotalBytes: 600,
		Partitions: []serverPartition{{Mountpoint: "/", UsedBytes: 100, TotalBytes: 200, UsedPercent: 50}},
	}
}

func insertMetric(t *testing.T, database *sql.DB, table string, resolutionMinutes int, metric sample) {
	t.Helper()
	if table == "deployment_metrics" {
		if _, err := database.Exec(`INSERT INTO deployment_metrics (
			deployment_id, sampled_at, container_id, container_name, state, cpu_percent, memory_used_bytes,
			memory_limit_bytes, network_in_bytes_total, network_out_bytes_total, uptime_seconds
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, metric.DeploymentID, metric.SampledAt, metric.ContainerID,
			metric.ContainerName, metric.State, metric.CPUPercent, metric.MemoryUsedBytes, metric.MemoryLimitBytes,
			metric.NetworkInBytesTotal, metric.NetworkOutBytesTotal, metric.UptimeSeconds); err != nil {
			t.Fatal(err)
		}
		return
	}
	if _, err := database.Exec(`INSERT INTO deployment_metric_rollups (
		deployment_id, resolution_minutes, sampled_at, container_id, container_name, state, cpu_percent,
		memory_used_bytes, memory_limit_bytes, network_in_bytes_total, network_out_bytes_total, uptime_seconds
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, metric.DeploymentID, resolutionMinutes, metric.SampledAt,
		metric.ContainerID, metric.ContainerName, metric.State, metric.CPUPercent, metric.MemoryUsedBytes,
		metric.MemoryLimitBytes, metric.NetworkInBytesTotal, metric.NetworkOutBytesTotal, metric.UptimeSeconds); err != nil {
		t.Fatal(err)
	}
}

func insertServerMetric(t *testing.T, database *sql.DB, table string, resolutionMinutes int, metric serverSample) {
	t.Helper()
	partitions, err := json.Marshal(metric.Partitions)
	if err != nil {
		t.Fatal(err)
	}
	if table == "server_metrics" {
		_, err = database.Exec(`INSERT INTO server_metrics (
			sampled_at, disk_used_bytes, disk_total_bytes, disk_used_percent, disk_read_bytes_total,
			disk_write_bytes_total, load_1, load_5, load_15, swap_used_bytes, swap_total_bytes, partitions_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, metric.SampledAt, metric.DiskUsedBytes, metric.DiskTotalBytes,
			metric.DiskUsedPercent, metric.DiskReadBytesTotal, metric.DiskWriteBytesTotal, metric.Load1, metric.Load5,
			metric.Load15, metric.SwapUsedBytes, metric.SwapTotalBytes, string(partitions))
	} else {
		_, err = database.Exec(`INSERT INTO server_metric_rollups (
			resolution_minutes, sampled_at, disk_used_bytes, disk_total_bytes, disk_used_percent, disk_read_bytes_total,
			disk_write_bytes_total, load_1, load_5, load_15, swap_used_bytes, swap_total_bytes, partitions_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, resolutionMinutes, metric.SampledAt, metric.DiskUsedBytes,
			metric.DiskTotalBytes, metric.DiskUsedPercent, metric.DiskReadBytesTotal, metric.DiskWriteBytesTotal,
			metric.Load1, metric.Load5, metric.Load15, metric.SwapUsedBytes, metric.SwapTotalBytes, string(partitions))
	}
	if err != nil {
		t.Fatal(err)
	}
}
