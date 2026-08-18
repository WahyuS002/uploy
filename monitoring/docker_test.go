package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeDockerAPI struct {
	containers  []dockerContainer
	inspections map[string]dockerInspect
	stats       map[string]dockerStats
	podmanRun   bool

	mu          sync.Mutex
	activeCalls int
	maxCalls    int
	delay       time.Duration
}

func (f *fakeDockerAPI) enter() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.activeCalls++
	if f.activeCalls > f.maxCalls {
		f.maxCalls = f.activeCalls
	}
}

func (f *fakeDockerAPI) leave() {
	f.mu.Lock()
	f.activeCalls--
	f.mu.Unlock()
}

func (f *fakeDockerAPI) listContainers(context.Context) ([]dockerContainer, error) {
	return f.containers, nil
}

func (f *fakeDockerAPI) inspectContainer(_ context.Context, id string) (dockerInspect, error) {
	f.enter()
	defer f.leave()
	time.Sleep(f.delay)
	return f.inspections[id], nil
}

func (f *fakeDockerAPI) containerStats(_ context.Context, id string) (dockerStats, error) {
	f.enter()
	defer f.leave()
	time.Sleep(f.delay)
	return f.stats[id], nil
}

func (f *fakeDockerAPI) podman() bool { return f.podmanRun }

func testDockerStats(read time.Time, cpu, system, preCPU, preSystem, memory, inactive, limit, rx, tx uint64) dockerStats {
	stats := dockerStats{Read: read.Format(time.RFC3339Nano), Networks: map[string]dockerNetwork{"eth0": {RxBytes: rx, TxBytes: tx}}}
	stats.CPUStats.CPUUsage.TotalUsage = cpu
	stats.CPUStats.SystemUsage = system
	stats.CPUStats.OnlineCPUs = 2
	stats.CPUStats.CPUUsage.Percpu = []uint64{cpu / 2, cpu / 2}
	stats.PreCPUStats.CPUUsage.TotalUsage = preCPU
	stats.PreCPUStats.SystemUsage = preSystem
	stats.MemoryStats.Usage = memory
	stats.MemoryStats.Stats.InactiveFile = inactive
	stats.MemoryStats.Limit = limit
	return stats
}

func TestCollectDockerSamplesUsesStructuredStats(t *testing.T) {
	now := time.Date(2026, time.August, 18, 12, 0, 15, 0, time.UTC)
	fake := &fakeDockerAPI{
		containers: []dockerContainer{
			{ID: "running-id", Names: []string{"/api"}, State: "running", Labels: map[string]string{"uploy.deployment_id": "deployment-a"}},
			{ID: "stopped-id", Names: []string{"worker"}, State: "exited", Labels: map[string]string{"uploy.deployment_id": "deployment-b"}},
			{ID: "ignored-id", State: "running", Labels: map[string]string{"other": "value"}},
		},
		inspections: map[string]dockerInspect{"running-id": {State: struct {
			Status    string `json:"Status"`
			StartedAt string `json:"StartedAt"`
		}{Status: "running", StartedAt: now.Add(-10 * time.Second).Format(time.RFC3339Nano)}}},
		stats: map[string]dockerStats{"running-id": testDockerStats(now, 300, 2_000, 100, 1_000, 1_000, 100, 10_000, 500, 700)},
	}
	tracker := newDockerMetricTracker()
	got, err := collectDockerSamples(context.Background(), fake, tracker, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d samples, want 2: %+v", len(got), got)
	}
	if got[0].DeploymentID != "deployment-a" || got[0].ContainerName != "api" {
		t.Fatalf("unexpected running sample: %+v", got[0])
	}
	if got[0].CPUPercent != 40 || got[0].MemoryUsedBytes != 900 || got[0].MemoryLimitBytes != 10_000 {
		t.Fatalf("structured stats = %+v", got[0])
	}
	if got[0].NetworkInBytesTotal != 500 || got[0].NetworkOutBytesTotal != 700 || got[0].UptimeSeconds != 10 {
		t.Fatalf("structured counters = %+v", got[0])
	}
	if got[1].State != "exited" || got[1].CPUPercent != 0 {
		t.Fatalf("stopped sample = %+v", got[1])
	}
	if fake.maxCalls > dockerRequestConcurrency {
		t.Fatalf("max concurrent Docker calls = %d, want <= %d", fake.maxCalls, dockerRequestConcurrency)
	}
}

func TestDockerMetricTrackerRejectsBadDeltasAndHandlesReset(t *testing.T) {
	tracker := newDockerMetricTracker()
	base := time.Unix(100, 0).UTC()
	first := testDockerStats(base, 1_000, 2_000, 0, 0, 100, 0, 1_000, 100, 100)
	if cpu, in, out := tracker.apply("id", first, false, base); cpu != 0 || in != 100 || out != 100 {
		t.Fatalf("first sample = cpu %v, network %d/%d", cpu, in, out)
	}
	bad := testDockerStats(base.Add(time.Second), 1_100, 2_100, 0, 0, 100, 0, 1_000, 10_000_000_000, 100)
	if _, in, out := tracker.apply("id", bad, false, base.Add(time.Second)); in != 100 || out != 100 {
		t.Fatalf("bad network delta = %d/%d, want previous totals", in, out)
	}
	reset := testDockerStats(base.Add(2*time.Second), 10, 20, 0, 0, 100, 0, 1_000, 5, 7)
	if cpu, in, out := tracker.apply("id", reset, false, base.Add(2*time.Second)); cpu != 0 || in != 5 || out != 7 {
		t.Fatalf("reset sample = cpu %v, network %d/%d", cpu, in, out)
	}

	ordered := newDockerMetricTracker()
	ordered.apply("id", testDockerStats(base, 100, 200, 0, 0, 100, 0, 1_000, 100, 100), false, base)
	ordered.apply("id", testDockerStats(base.Add(2*time.Second), 200, 400, 0, 0, 100, 0, 1_000, 200, 200), false, base.Add(2*time.Second))
	if cpu, in, out := ordered.apply("id", testDockerStats(base.Add(time.Second), 150, 300, 0, 0, 100, 0, 1_000, 150, 150), false, base.Add(time.Second)); cpu != 0 || in != 200 || out != 200 {
		t.Fatalf("out-of-order sample = cpu %v, network %d/%d", cpu, in, out)
	}
}

func TestCollectDockerSamplesBoundsConcurrency(t *testing.T) {
	now := time.Date(2026, time.August, 18, 12, 0, 0, 0, time.UTC)
	fake := &fakeDockerAPI{inspections: make(map[string]dockerInspect), stats: make(map[string]dockerStats), delay: 10 * time.Millisecond}
	for index := 0; index < dockerRequestConcurrency+2; index++ {
		id := string(rune('a' + index))
		fake.containers = append(fake.containers, dockerContainer{ID: id, State: "running", Labels: map[string]string{"uploy.deployment_id": id}})
		fake.stats[id] = testDockerStats(now, 100, 200, 0, 0, 100, 0, 1_000, 1, 1)
	}
	if _, err := collectDockerSamples(context.Background(), fake, newDockerMetricTracker(), now); err != nil {
		t.Fatal(err)
	}
	if fake.maxCalls != dockerRequestConcurrency {
		t.Fatalf("max concurrent Docker calls = %d, want %d", fake.maxCalls, dockerRequestConcurrency)
	}
}

func TestDockerMetricTrackerCalculatesPodmanCPU(t *testing.T) {
	tracker := newDockerMetricTracker()
	base := time.Unix(100, 0).UTC()
	first := testDockerStats(base, 1_000_000_000, 0, 0, 0, 100, 0, 1_000, 1, 1)
	if cpu, _, _ := tracker.apply("id", first, true, base); cpu != 0 {
		t.Fatalf("first Podman CPU = %v, want 0", cpu)
	}
	second := testDockerStats(base.Add(time.Second), 2_000_000_000, 0, 0, 0, 100, 0, 1_000, 2, 2)
	if cpu, _, _ := tracker.apply("id", second, true, base.Add(time.Second)); cpu != 50 {
		t.Fatalf("Podman CPU = %v, want 50", cpu)
	}
}

func TestMemoryUsageRejectsMalformedValues(t *testing.T) {
	stats := dockerStats{}
	stats.MemoryStats.Usage = 10
	stats.MemoryStats.Stats.InactiveFile = 11
	if _, _, ok := memoryUsage(stats); ok {
		t.Fatal("expected underflowing memory stats to be rejected")
	}
	stats.MemoryStats.Stats.InactiveFile = 0
	stats.MemoryStats.Usage = maxMemoryUsage + 1
	if _, _, ok := memoryUsage(stats); ok {
		t.Fatal("expected oversized memory stats to be rejected")
	}
}

func TestDockerClientUsesVersionedHTTPAPI(t *testing.T) {
	client := newDockerClient("unused")
	client.httpClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		newResponse := func(status int, body string) *http.Response {
			return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
		}
		switch request.URL.Path {
		case "/version":
			response := newResponse(http.StatusOK, `{"ApiVersion":"1.44","Components":[{"Name":"Engine"}]}`)
			response.Header.Set("Server", "Docker/27")
			return response, nil
		case "/v1.44/containers/json":
			if request.URL.Query().Get("all") != "1" {
				return newResponse(http.StatusBadRequest, "missing all"), nil
			}
			var filters map[string][]string
			if err := json.Unmarshal([]byte(request.URL.Query().Get("filters")), &filters); err != nil || len(filters["label"]) != 1 || filters["label"][0] != "uploy.deployment_id" {
				return newResponse(http.StatusBadRequest, "missing label filter"), nil
			}
			return newResponse(http.StatusOK, `[{"Id":"id","Names":["/api"],"State":"running","Labels":{"uploy.deployment_id":"deployment"}}]`), nil
		case "/v1.44/containers/id/json":
			return newResponse(http.StatusOK, `{}`), nil
		case "/v1.44/containers/id/stats":
			return newResponse(http.StatusOK, `{}`), nil
		default:
			return newResponse(http.StatusNotFound, "not found"), nil
		}
	})}
	if _, err := client.listContainers(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := client.inspectContainer(context.Background(), "id"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.containerStats(context.Background(), "id"); err != nil {
		t.Fatal(err)
	}
	if client.podman() {
		t.Fatal("Docker server detected as Podman")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }
