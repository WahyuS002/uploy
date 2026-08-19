package monitoring

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	dockerapi "github.com/WahyuS002/uploy/docker"
	"github.com/WahyuS002/uploy/proxy"
	"github.com/WahyuS002/uploy/ssh"
)

const (
	ContainerName        = "uploy-monitor"
	AgentPort            = 9184
	loopbackAddress      = "127.0.0.1"
	dataDir              = "/data/uploy/monitoring"
	enableCleanupTimeout = 30 * time.Second
)

type Config struct {
	Image string
	// HostPort is the loopback port the agent is published on. It is
	// configurable only so a server that already uses the default can move it
	// out of the way; nothing outside the machine can reach it either way.
	HostPort      int
	RetentionDays int
	FQDN          string
	ControlToken  string
	ReaderToken   string
}

type HistoryPoint struct {
	DeploymentID         string  `json:"deployment_id"`
	ContainerID          string  `json:"container_id"`
	ContainerName        string  `json:"container_name"`
	State                string  `json:"state"`
	SampledAt            int64   `json:"sampled_at"`
	CPUPercent           float64 `json:"cpu_percent"`
	MemoryUsedBytes      int64   `json:"memory_used_bytes"`
	MemoryLimitBytes     int64   `json:"memory_limit_bytes"`
	NetworkInBytesTotal  int64   `json:"network_in_bytes_total"`
	NetworkOutBytesTotal int64   `json:"network_out_bytes_total"`
	UptimeSeconds        int64   `json:"uptime_seconds"`
}

type HistoryResponse struct {
	Points []HistoryPoint `json:"points"`
}

type HistoriesResponse struct {
	Deployments map[string]HistoryResponse `json:"deployments"`
}

type LatestResponse struct {
	Points []HistoryPoint `json:"points"`
}

type ServerLatestResponse struct {
	SampledAt           int64             `json:"sampled_at"`
	DiskUsedBytes       int64             `json:"disk_used_bytes"`
	DiskTotalBytes      int64             `json:"disk_total_bytes"`
	DiskUsedPercent     float64           `json:"disk_used_percent"`
	DiskReadBytesTotal  int64             `json:"disk_read_bytes_total"`
	DiskWriteBytesTotal int64             `json:"disk_write_bytes_total"`
	Load1               float64           `json:"load_1"`
	Load5               float64           `json:"load_5"`
	Load15              float64           `json:"load_15"`
	SwapUsedBytes       int64             `json:"swap_used_bytes"`
	SwapTotalBytes      int64             `json:"swap_total_bytes"`
	Partitions          []ServerPartition `json:"partitions"`
}

type ServerPartition struct {
	Mountpoint  string  `json:"mountpoint"`
	UsedBytes   int64   `json:"used_bytes"`
	TotalBytes  int64   `json:"total_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

type ServerHistoryResponse struct {
	Points []ServerLatestResponse `json:"points"`
}

func Enable(ctx context.Context, client *ssh.Client, serverID string, cfg Config) (err error) {
	if err := ValidateConfig(cfg); err != nil {
		return err
	}

	if err := prepareMonitoringHost(ctx, client, cfg); err != nil {
		return err
	}

	docker := client.DockerBin()
	rollbackName := ContainerName + "-rollback"
	_, _ = client.Run(ctx, docker+" rm -f "+rollbackName+" >/dev/null 2>&1 || true")
	hadOld, err := dockerapi.ContainerExists(ctx, client, ContainerName)
	if err != nil {
		return fmt.Errorf("inspect existing monitoring container: %w", err)
	}

	state := enableState{
		hadOld:       hadOld,
		rollbackName: rollbackName,
	}
	defer func() {
		if cleanupErr := state.cleanup(ctx, client, err == nil); cleanupErr != nil {
			err = errors.Join(err, cleanupErr)
		}
	}()

	if err := state.prepareOld(ctx, client); err != nil {
		return err
	}

	if out, err := client.Run(ctx, buildRunCommand(docker, cfg)); err != nil {
		if trimmed := strings.TrimSpace(out); trimmed != "" {
			return fmt.Errorf("start monitoring agent: %s", trimmed)
		}
		return fmt.Errorf("start monitoring agent: %w", err)
	}
	state.newStarted = true
	if err := waitHealthy(ctx, client); err != nil {
		return err
	}

	if err := syncRoute(client, serverID, cfg.FQDN); err != nil {
		return err
	}
	return nil
}

func prepareMonitoringHost(ctx context.Context, client *ssh.Client, cfg Config) error {
	if err := proxy.EnsureNetwork(client); err != nil {
		return err
	}
	if cfg.FQDN != "" {
		if err := proxy.EnsureProxy(client, nil); err != nil {
			return fmt.Errorf("prepare monitoring public route: %w", err)
		}
	}
	docker := client.DockerBin()
	if out, err := client.Run(ctx, docker+" pull "+ssh.ShellQuote(cfg.Image)); err != nil {
		if trimmed := strings.TrimSpace(out); trimmed != "" {
			return fmt.Errorf("pull monitoring image: %s", trimmed)
		}
		return fmt.Errorf("pull monitoring image: %w", err)
	}
	if out, err := client.RunElevated(ctx, "mkdir -p "+dataDir); err != nil {
		if trimmed := strings.TrimSpace(out); trimmed != "" {
			return fmt.Errorf("prepare monitoring data: %s", trimmed)
		}
		return fmt.Errorf("prepare monitoring data: %w", err)
	}
	return nil
}

// buildRunCommand publishes the agent on loopback only. The control plane
// reaches it by tunneling through the server's SSH connection, and a public
// FQDN reaches it over the uploy network by container name, so binding the port
// to a routable address would expose it without anything asking for that.
func buildRunCommand(docker string, cfg Config) string {
	publishedAddress := net.JoinHostPort(loopbackAddress, strconv.Itoa(cfg.HostPort))
	return fmt.Sprintf(
		"%s run -d --name %s --restart unless-stopped --network uploy -p %s:%d "+
			"-v /var/run/docker.sock:/var/run/docker.sock:ro -v /proc:/host/proc:ro -v /sys:/host/sys:ro -v /:/host:ro -v %s:/data "+
			"-e UPLOY_MONITOR_CONTROL_TOKEN=%s -e UPLOY_MONITOR_READER_TOKEN=%s "+
			"-e UPLOY_MONITOR_RETENTION_DAYS=%d -e HOST_PROC=/host/proc -e HOST_SYS=/host/sys -e HOST_ROOT=/host %s",
		docker, ContainerName, publishedAddress, AgentPort, dataDir,
		ssh.ShellQuote(cfg.ControlToken), ssh.ShellQuote(cfg.ReaderToken), cfg.RetentionDays, ssh.ShellQuote(cfg.Image),
	)
}

func syncRoute(client *ssh.Client, serverID, fqdn string) error {
	id := routeID(serverID)
	if fqdn == "" {
		if err := proxy.RemoveRoute(client, id); err != nil {
			return fmt.Errorf("remove monitoring public route: %w", err)
		}
		return nil
	}
	if err := proxy.SetMonitoringRoute(client, id, []string{fqdn}, ContainerName, AgentPort); err != nil {
		return fmt.Errorf("publish monitoring public route: %w", err)
	}
	return nil
}

type enableState struct {
	hadOld        bool
	oldStopped    bool
	oldMoved      bool
	oldWasRunning bool
	newStarted    bool
	rollbackName  string
}

func (s *enableState) prepareOld(ctx context.Context, client dockerapi.CommandRunner) error {
	if !s.hadOld {
		return nil
	}
	running, err := dockerapi.ContainerRunning(ctx, client, ContainerName)
	if err != nil {
		return fmt.Errorf("inspect existing monitoring state: %w", err)
	}
	s.oldWasRunning = running
	docker := client.DockerBin()
	if running {
		if out, err := client.Run(ctx, docker+" stop "+ContainerName); err != nil {
			if trimmed := strings.TrimSpace(out); trimmed != "" {
				return fmt.Errorf("stop existing monitoring agent: %s", trimmed)
			}
			return fmt.Errorf("stop existing monitoring agent: %w", err)
		}
		s.oldStopped = true
	}
	if out, err := client.Run(ctx, docker+" rename "+ContainerName+" "+s.rollbackName); err != nil {
		if trimmed := strings.TrimSpace(out); trimmed != "" {
			return fmt.Errorf("rename existing monitoring agent: %s", trimmed)
		}
		return fmt.Errorf("rename existing monitoring agent: %w", err)
	}
	s.oldMoved = true
	return nil
}

func (s *enableState) cleanup(parent context.Context, client dockerapi.CommandRunner, committed bool) error {
	if !s.oldMoved && !s.oldStopped && !s.newStarted {
		return nil
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(parent), enableCleanupTimeout)
	defer cancel()
	docker := client.DockerBin()

	var cleanupErr error
	if committed {
		if s.oldMoved {
			cleanupErr = errors.Join(cleanupErr, removeContainerIfPresent(cleanupCtx, client, s.rollbackName))
		}
		return cleanupErr
	}

	if s.newStarted || s.oldMoved {
		cleanupErr = errors.Join(cleanupErr, removeContainerIfPresent(cleanupCtx, client, ContainerName))
	}
	if s.oldMoved {
		restore := docker + " rename " + s.rollbackName + " " + ContainerName
		if s.oldWasRunning {
			restore += " && " + docker + " start " + ContainerName
		}
		_, err := client.Run(cleanupCtx, restore)
		cleanupErr = errors.Join(cleanupErr, err)
	} else if s.oldStopped {
		_, err := client.Run(cleanupCtx, docker+" start "+ContainerName)
		cleanupErr = errors.Join(cleanupErr, err)
	}
	return cleanupErr
}

func removeContainerIfPresent(ctx context.Context, client dockerapi.CommandRunner, name string) error {
	exists, err := dockerapi.ContainerExists(ctx, client, name)
	if err != nil || !exists {
		return err
	}
	_, err = client.Run(ctx, client.DockerBin()+" rm -f "+ssh.ShellQuote(name))
	return err
}

func ValidateConfig(cfg Config) error {
	if !pinnedImage(cfg.Image) {
		return errors.New("monitoring image must use a pinned tag or digest")
	}
	if cfg.HostPort < 1 || cfg.HostPort > 65535 {
		return errors.New("monitoring port is invalid")
	}
	if cfg.RetentionDays < 1 || cfg.RetentionDays > 30 {
		return errors.New("monitoring retention must be between 1 and 30 days")
	}
	if cfg.ControlToken == "" || cfg.ReaderToken == "" {
		return errors.New("monitoring control and reader tokens are required")
	}
	return nil
}

func Disable(ctx context.Context, client *ssh.Client, serverID string) error {
	if err := proxy.RemoveRoute(client, routeID(serverID)); err != nil {
		return fmt.Errorf("remove monitoring route: %w", err)
	}
	docker := client.DockerBin()
	_, err := client.Run(ctx, docker+" rm -f "+ContainerName+" "+ContainerName+"-rollback >/dev/null 2>&1 || true")
	return err
}

func DeleteLocalData(ctx context.Context, client *ssh.Client) error {
	_, err := client.RunElevated(ctx, "rm -rf "+dataDir)
	return err
}

// Timeouts are per call rather than per transport: a read of the current sample
// sits in front of a polling dashboard and should give up quickly, while a read
// that scans stored samples is allowed to take its time.
const (
	liveTimeout    = 5 * time.Second
	historyTimeout = 15 * time.Second
)

// Response body limits. The agent is trusted but not unbounded — a runaway
// history query should fail rather than exhaust the control plane's memory.
const (
	historyLimit      = 16 << 20
	latestLimit       = 8 << 20
	serverLatestLimit = 1 << 20
)

func GetHistories(ctx context.Context, target Target, controlToken string, deploymentIDs []string, from, to time.Time, maxPoints int) (HistoriesResponse, error) {
	if len(deploymentIDs) == 0 {
		return HistoriesResponse{Deployments: map[string]HistoryResponse{}}, nil
	}
	payload, err := json.Marshal(struct {
		DeploymentIDs []string `json:"deployment_ids"`
		From          string   `json:"from"`
		To            string   `json:"to"`
		MaxPoints     int      `json:"max_points"`
	}{
		DeploymentIDs: deploymentIDs,
		From:          from.UTC().Format(time.RFC3339),
		To:            to.UTC().Format(time.RFC3339),
		MaxPoints:     maxPoints,
	})
	if err != nil {
		return HistoriesResponse{}, err
	}
	histories, err := withTransport(target, func(tr Transport) (HistoriesResponse, error) {
		var histories HistoriesResponse
		err := fetch(ctx, tr, agentCall{
			method:  http.MethodPost,
			path:    "/v1/history",
			token:   controlToken,
			payload: payload,
			timeout: historyTimeout,
			limit:   historyLimit,
		}, &histories)
		return histories, err
	})
	if err != nil {
		return HistoriesResponse{}, err
	}
	if histories.Deployments == nil {
		histories.Deployments = map[string]HistoryResponse{}
	}
	return histories, nil
}

func GetLatestAll(ctx context.Context, target Target, controlToken string) (LatestResponse, error) {
	latest, err := withTransport(target, func(tr Transport) (LatestResponse, error) {
		var latest LatestResponse
		err := fetch(ctx, tr, agentCall{
			method:  http.MethodGet,
			path:    "/v1/latest",
			token:   controlToken,
			timeout: liveTimeout,
			limit:   latestLimit,
		}, &latest)
		return latest, err
	})
	if err != nil {
		return LatestResponse{}, err
	}
	if latest.Points == nil {
		latest.Points = []HistoryPoint{}
	}
	return latest, nil
}

func GetServerLatest(ctx context.Context, target Target, controlToken string) (ServerLatestResponse, error) {
	return withTransport(target, func(tr Transport) (ServerLatestResponse, error) {
		var latest ServerLatestResponse
		err := fetch(ctx, tr, agentCall{
			method:  http.MethodGet,
			path:    "/v1/server/latest",
			token:   controlToken,
			timeout: liveTimeout,
			limit:   serverLatestLimit,
		}, &latest)
		return latest, err
	})
}

func GetServerHistory(ctx context.Context, target Target, controlToken string, from, to time.Time, maxPoints int) (ServerHistoryResponse, error) {
	path := "/v1/server/history?from=" + url.QueryEscape(from.UTC().Format(time.RFC3339)) +
		"&to=" + url.QueryEscape(to.UTC().Format(time.RFC3339)) +
		"&max_points=" + strconv.Itoa(maxPoints)
	return withTransport(target, func(tr Transport) (ServerHistoryResponse, error) {
		var history ServerHistoryResponse
		err := fetch(ctx, tr, agentCall{
			method:  http.MethodGet,
			path:    path,
			token:   controlToken,
			timeout: historyTimeout,
			limit:   historyLimit,
		}, &history)
		return history, err
	})
}

func DeleteHistory(ctx context.Context, target Target, controlToken string) error {
	_, err := withTransport(target, func(tr Transport) (struct{}, error) {
		return struct{}{}, fetch(ctx, tr, agentCall{
			method:  http.MethodDelete,
			path:    "/v1/history",
			token:   controlToken,
			timeout: historyTimeout,
		}, nil)
	})
	return err
}

// agentCall describes one request to the agent.
type agentCall struct {
	method  string
	path    string
	token   string
	payload []byte
	timeout time.Duration
	// limit caps how much of the response body is decoded. Ignored when the
	// call expects no content.
	limit int64
}

// fetch sends call and decodes its JSON response into dest. A nil dest means the
// endpoint answers 204 with no body.
func fetch(ctx context.Context, tr Transport, call agentCall, dest any) error {
	ctx, cancel := context.WithTimeout(ctx, call.timeout)
	defer cancel()

	var body io.Reader
	if call.payload != nil {
		body = bytes.NewReader(call.payload)
	}
	req, err := http.NewRequestWithContext(ctx, call.method, tr.BaseURL()+call.path, body)
	if err != nil {
		return err
	}
	if call.payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+call.token)

	response, err := tr.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if dest == nil {
		if response.StatusCode != http.StatusNoContent {
			return agentStatusError(response)
		}
		return nil
	}
	if response.StatusCode != http.StatusOK {
		return agentStatusError(response)
	}
	return json.NewDecoder(io.LimitReader(response.Body, call.limit)).Decode(dest)
}

// agentStatusError quotes whatever the agent said about the failure. Its own
// message is usually the half worth reading.
func agentStatusError(response *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	if detail := strings.TrimSpace(string(body)); detail != "" {
		return fmt.Errorf("monitoring agent HTTP %d: %s", response.StatusCode, detail)
	}
	return fmt.Errorf("monitoring agent HTTP %d", response.StatusCode)
}

func routeID(serverID string) string { return "monitoring-" + serverID }

func pinnedImage(image string) bool {
	if strings.Contains(image, "@sha256:") {
		return true
	}
	colon := strings.LastIndex(image, ":")
	slash := strings.LastIndex(image, "/")
	return colon > slash && image[colon+1:] != "latest"
}

func waitHealthy(ctx context.Context, client *ssh.Client) error {
	deadline, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {
		status, err := dockerapi.ContainerHealth(deadline, client, ContainerName)
		if err == nil && status == "healthy" {
			return nil
		}
		select {
		case <-deadline.Done():
			return errors.New("monitoring agent failed its health check")
		case <-time.After(500 * time.Millisecond):
		}
	}
}
