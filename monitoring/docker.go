package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	dockerRequestConcurrency = 5
	dockerRequestTimeout     = 10 * time.Second
	maxDockerResponseBytes   = 8 << 20
	maxNetworkSpeedBps       = 5e9
	maxMemoryUsage           = uint64(100) * 1024 * 1024 * 1024 * 1024
)

type dockerAPI interface {
	listContainers(context.Context) ([]dockerContainer, error)
	inspectContainer(context.Context, string) (dockerInspect, error)
	containerStats(context.Context, string) (dockerStats, error)
	podman() bool
}

type dockerClient struct {
	httpClient *http.Client
	semaphore  chan struct{}

	versionMu    sync.RWMutex
	versionReady bool
	apiVersion   string
	usingPodman  bool
}

func newDockerClient(socketPath string) *dockerClient {
	transport := &http.Transport{
		DisableKeepAlives: true,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
	return &dockerClient{
		httpClient: &http.Client{Transport: transport, Timeout: dockerRequestTimeout},
		semaphore:  make(chan struct{}, dockerRequestConcurrency),
	}
}

type dockerContainer struct {
	ID     string            `json:"Id"`
	Names  []string          `json:"Names"`
	State  string            `json:"State"`
	Status string            `json:"Status"`
	Labels map[string]string `json:"Labels"`
}

type dockerInspect struct {
	ID    string `json:"Id"`
	Name  string `json:"Name"`
	State struct {
		Status    string `json:"Status"`
		StartedAt string `json:"StartedAt"`
	} `json:"State"`
}

type dockerStats struct {
	Read        string                   `json:"read"`
	CPUStats    dockerCPUStats           `json:"cpu_stats"`
	PreCPUStats dockerCPUStats           `json:"precpu_stats"`
	MemoryStats dockerMemoryStats        `json:"memory_stats"`
	Networks    map[string]dockerNetwork `json:"networks"`
}

type dockerCPUStats struct {
	CPUUsage struct {
		TotalUsage uint64   `json:"total_usage"`
		Percpu     []uint64 `json:"percpu_usage"`
	} `json:"cpu_usage"`
	SystemUsage uint64 `json:"system_cpu_usage"`
	OnlineCPUs  uint32 `json:"online_cpus"`
}

type dockerMemoryStats struct {
	Usage uint64 `json:"usage"`
	Limit uint64 `json:"limit"`
	Stats struct {
		Cache        uint64 `json:"cache"`
		InactiveFile uint64 `json:"inactive_file"`
	} `json:"stats"`
	PrivateWorkingSet uint64 `json:"privateworkingset"`
}

type dockerNetwork struct {
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
}

type dockerVersion struct {
	APIVersion string `json:"ApiVersion"`
	Components []struct {
		Name string `json:"Name"`
	} `json:"Components"`
}

func (c *dockerClient) podman() bool {
	c.versionMu.RLock()
	defer c.versionMu.RUnlock()
	return c.usingPodman
}

func (c *dockerClient) listContainers(ctx context.Context) ([]dockerContainer, error) {
	_ = c.ensureVersion(ctx)
	filters, err := json.Marshal(map[string][]string{"label": {"uploy.deployment_id"}})
	if err != nil {
		return nil, err
	}
	path := "/containers/json?all=1&filters=" + url.QueryEscape(string(filters))
	var containers []dockerContainer
	if err := c.getJSON(ctx, path, &containers); err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}
	return containers, nil
}

func (c *dockerClient) inspectContainer(ctx context.Context, id string) (dockerInspect, error) {
	_ = c.ensureVersion(ctx)
	var inspection dockerInspect
	if err := c.getJSON(ctx, "/containers/"+url.PathEscape(id)+"/json", &inspection); err != nil {
		return dockerInspect{}, fmt.Errorf("inspect container %s: %w", id, err)
	}
	return inspection, nil
}

func (c *dockerClient) containerStats(ctx context.Context, id string) (dockerStats, error) {
	_ = c.ensureVersion(ctx)
	var stats dockerStats
	if err := c.getJSON(ctx, "/containers/"+url.PathEscape(id)+"/stats?stream=0&one-shot=1", &stats); err != nil {
		return dockerStats{}, fmt.Errorf("read stats for container %s: %w", id, err)
	}
	return stats, nil
}

func (c *dockerClient) ensureVersion(ctx context.Context) error {
	c.versionMu.RLock()
	ready := c.versionReady
	c.versionMu.RUnlock()
	if ready {
		return nil
	}

	body, headers, status, err := c.do(ctx, "/version")
	if err != nil {
		return err
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return fmt.Errorf("version endpoint returned HTTP %d", status)
	}
	var version dockerVersion
	if err := decodeDockerJSON(body, &version); err != nil {
		return fmt.Errorf("decode Docker version: %w", err)
	}
	usingPodman := detectPodmanHeader(headers.Get("Server"))
	for _, component := range version.Components {
		if strings.Contains(strings.ToLower(component.Name), "podman") {
			usingPodman = true
		}
	}
	c.versionMu.Lock()
	c.apiVersion = strings.TrimPrefix(strings.TrimSpace(version.APIVersion), "v")
	c.usingPodman = usingPodman
	c.versionReady = true
	c.versionMu.Unlock()
	return nil
}

func detectPodmanHeader(server string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(server)), "libpod")
}

func (c *dockerClient) apiPrefix() string {
	c.versionMu.RLock()
	defer c.versionMu.RUnlock()
	if c.apiVersion == "" {
		return ""
	}
	return "/v" + c.apiVersion
}

func (c *dockerClient) getJSON(ctx context.Context, path string, target any) error {
	prefix := c.apiPrefix()
	body, headers, status, err := c.do(ctx, prefix+path)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound && prefix != "" {
		body, headers, status, err = c.do(ctx, path)
		if err != nil {
			return err
		}
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		message := strings.TrimSpace(string(body))
		if len(message) > 512 {
			message = message[:512]
		}
		return fmt.Errorf("Docker API returned HTTP %d: %s", status, message)
	}
	if detectPodmanHeader(headers.Get("Server")) {
		c.versionMu.Lock()
		c.usingPodman = true
		c.versionMu.Unlock()
	}
	return decodeDockerJSON(body, target)
}

func (c *dockerClient) do(ctx context.Context, path string) ([]byte, http.Header, int, error) {
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return nil, nil, 0, ctx.Err()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://docker"+path, nil)
	if err != nil {
		return nil, nil, 0, err
	}
	req.Header.Set("User-Agent", "uploy-monitor")
	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, 0, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxDockerResponseBytes+1))
	if err != nil {
		return nil, response.Header, response.StatusCode, err
	}
	if len(body) > maxDockerResponseBytes {
		return nil, response.Header, response.StatusCode, fmt.Errorf("Docker API response exceeds %d bytes", maxDockerResponseBytes)
	}
	return body, response.Header, response.StatusCode, nil
}

func decodeDockerJSON(body []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

type dockerMetricState struct {
	cpuReadAt, networkReadAt time.Time
	cpuTotal, cpuSystem      uint64
	networkIn, networkOut    uint64
	hasCPU, hasNetwork       bool
}

type dockerMetricTracker struct {
	mu         sync.Mutex
	containers map[string]dockerMetricState
}

func newDockerMetricTracker() *dockerMetricTracker {
	return &dockerMetricTracker{containers: make(map[string]dockerMetricState)}
}

func (t *dockerMetricTracker) prune(ids map[string]struct{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for id := range t.containers {
		if _, ok := ids[id]; !ok {
			delete(t.containers, id)
		}
	}
}

func (t *dockerMetricTracker) apply(id string, stats dockerStats, usingPodman bool, now time.Time) (float64, uint64, uint64) {
	readAt := now
	if parsed, err := time.Parse(time.RFC3339Nano, stats.Read); err == nil && !parsed.IsZero() {
		readAt = parsed
	}
	currentCPU := stats.CPUStats.CPUUsage.TotalUsage
	currentSystem := stats.CPUStats.SystemUsage
	networkIn, networkOut := networkTotals(stats.Networks)

	t.mu.Lock()
	defer t.mu.Unlock()
	state := t.containers[id]
	var cpuPercent float64
	if !state.hasCPU || readAt.After(state.cpuReadAt) {
		prevCPU, prevSystem := state.cpuTotal, state.cpuSystem
		if !state.hasCPU {
			prevCPU = stats.PreCPUStats.CPUUsage.TotalUsage
			prevSystem = stats.PreCPUStats.SystemUsage
		}
		if usingPodman {
			if state.hasCPU && currentCPU >= prevCPU && stats.CPUStats.OnlineCPUs > 0 {
				elapsed := readAt.Sub(state.cpuReadAt)
				if elapsed > 0 {
					cpuPercent = float64(currentCPU-prevCPU) / float64(elapsed.Nanoseconds()*int64(stats.CPUStats.OnlineCPUs)) * 100
				}
			}
		} else if currentCPU >= prevCPU && currentSystem >= prevSystem {
			systemDelta := currentSystem - prevSystem
			if systemDelta > 0 && prevCPU > 0 {
				onlineCPUs := stats.CPUStats.OnlineCPUs
				if onlineCPUs == 0 {
					onlineCPUs = uint32(len(stats.CPUStats.CPUUsage.Percpu))
				}
				if onlineCPUs == 0 {
					onlineCPUs = 1
				}
				cpuPercent = float64(currentCPU-prevCPU) / float64(systemDelta) * float64(onlineCPUs) * 100
			}
		}
		state.cpuTotal = currentCPU
		state.cpuSystem = currentSystem
		state.cpuReadAt = readAt
		state.hasCPU = true
	}

	if !state.hasNetwork || readAt.After(state.networkReadAt) {
		validNetwork := true
		if state.hasNetwork {
			if networkIn < state.networkIn || networkOut < state.networkOut {
				// A restarted container starts a fresh cumulative counter.
				state.networkIn, state.networkOut = networkIn, networkOut
			} else {
				elapsed := readAt.Sub(state.networkReadAt).Seconds()
				if elapsed > 0 && (float64(networkIn-state.networkIn)/elapsed > maxNetworkSpeedBps || float64(networkOut-state.networkOut)/elapsed > maxNetworkSpeedBps) {
					validNetwork = false
				}
				if validNetwork {
					state.networkIn, state.networkOut = networkIn, networkOut
				}
			}
		} else {
			state.networkIn, state.networkOut = networkIn, networkOut
		}
		if validNetwork {
			state.networkReadAt = readAt
			state.hasNetwork = true
		}
	}
	t.containers[id] = state
	if !state.hasNetwork {
		return cpuPercent, networkIn, networkOut
	}
	return cpuPercent, state.networkIn, state.networkOut
}

func networkTotals(networks map[string]dockerNetwork) (uint64, uint64) {
	var received, sent uint64
	for _, network := range networks {
		if math.MaxUint64-received < network.RxBytes {
			received = math.MaxUint64
		} else {
			received += network.RxBytes
		}
		if math.MaxUint64-sent < network.TxBytes {
			sent = math.MaxUint64
		} else {
			sent += network.TxBytes
		}
	}
	return received, sent
}

func memoryUsage(stats dockerStats) (int64, int64, bool) {
	if stats.MemoryStats.Usage > maxMemoryUsage {
		return 0, 0, false
	}
	cache := stats.MemoryStats.Stats.InactiveFile
	if cache == 0 {
		cache = stats.MemoryStats.Stats.Cache
	}
	if cache > stats.MemoryStats.Usage {
		return 0, 0, false
	}
	used := stats.MemoryStats.Usage - cache
	if used == 0 || used > maxMemoryUsage || stats.MemoryStats.Limit > uint64(math.MaxInt64) {
		return 0, 0, false
	}
	return int64(used), int64(stats.MemoryStats.Limit), true
}

func (c *collector) readDockerSamples(ctx context.Context) ([]sample, error) {
	if c.docker == nil {
		return []sample{}, nil
	}
	tracker := c.tracker
	if tracker == nil {
		tracker = newDockerMetricTracker()
		c.tracker = tracker
	}
	return collectDockerSamples(ctx, c.docker, tracker, time.Now().UTC())
}

type dockerContainerResult struct {
	id         string
	inspection dockerInspect
	inspectErr error
	stats      dockerStats
	statsErr   error
}

func collectDockerSamples(ctx context.Context, client dockerAPI, tracker *dockerMetricTracker, now time.Time) ([]sample, error) {
	containers, err := client.listContainers(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]sample, len(containers))
	running := make([]dockerContainer, 0, len(containers))
	observedIDs := make(map[string]struct{}, len(containers))
	for _, container := range containers {
		id := strings.TrimSpace(container.ID)
		deploymentID := strings.TrimSpace(container.Labels["uploy.deployment_id"])
		if id == "" || deploymentID == "" {
			continue
		}
		name := ""
		if len(container.Names) > 0 {
			name = strings.TrimPrefix(strings.TrimSpace(container.Names[0]), "/")
		}
		state := strings.TrimSpace(container.State)
		if state == "" {
			state = strings.TrimSpace(container.Status)
		}
		byID[id] = sample{DeploymentID: deploymentID, ContainerID: id, ContainerName: name, State: state, SampledAt: now.UnixMilli()}
		observedIDs[id] = struct{}{}
		if state == "running" {
			running = append(running, container)
		}
	}
	tracker.prune(observedIDs)
	if len(running) == 0 {
		return sortedSamples(byID), nil
	}

	jobs := make(chan dockerContainer)
	results := make(chan dockerContainerResult, len(running))
	workers := min(dockerRequestConcurrency, len(running))
	var waitGroup sync.WaitGroup
	for index := 0; index < workers; index++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for container := range jobs {
				inspection, inspectErr := client.inspectContainer(ctx, container.ID)
				stats, statsErr := client.containerStats(ctx, container.ID)
				results <- dockerContainerResult{id: container.ID, inspection: inspection, inspectErr: inspectErr, stats: stats, statsErr: statsErr}
			}
		}()
	}
	for _, container := range running {
		jobs <- container
	}
	close(jobs)
	waitGroup.Wait()
	close(results)

	for result := range results {
		metric, ok := byID[result.id]
		if !ok || result.statsErr != nil {
			continue
		}
		used, limit, validMemory := memoryUsage(result.stats)
		if !validMemory {
			continue
		}
		cpuPercent, networkIn, networkOut := tracker.apply(result.id, result.stats, client.podman(), now)
		metric.CPUPercent = cpuPercent
		metric.MemoryUsedBytes = used
		metric.MemoryLimitBytes = limit
		metric.NetworkInBytesTotal = safeInt64(networkIn)
		metric.NetworkOutBytesTotal = safeInt64(networkOut)
		if result.inspectErr == nil {
			if started, parseErr := time.Parse(time.RFC3339Nano, result.inspection.State.StartedAt); parseErr == nil {
				metric.UptimeSeconds = max(0, int64(now.Sub(started).Seconds()))
			}
		}
		byID[result.id] = metric
	}
	return sortedSamples(byID), nil
}
