package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/WahyuS002/uploy/proxy"
	"github.com/WahyuS002/uploy/ssh"
)

const (
	ContainerName = "uploy-monitor"
	AgentPort     = 9184
	dataDir       = "/data/uploy/monitoring"
)

type Config struct {
	Image          string
	PrivateAddress string
	HostPort       int
	RetentionDays  int
	FQDN           string
	ControlToken   string
	ReaderToken    string
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

func Enable(ctx context.Context, client *ssh.Client, serverID string, cfg Config) error {
	if err := ValidateConfig(cfg); err != nil {
		return err
	}

	if err := proxy.EnsureNetwork(client); err != nil {
		return err
	}
	if cfg.FQDN != "" {
		if err := proxy.EnsureProxy(client, nil); err != nil {
			return fmt.Errorf("prepare monitoring public route: %w", err)
		}
	}
	docker := client.DockerBin()
	if _, err := client.Run(ctx, docker+" pull "+ssh.ShellQuote(cfg.Image)); err != nil {
		return fmt.Errorf("pull monitoring image: %w", err)
	}
	if _, err := client.RunElevated(ctx, "mkdir -p "+dataDir); err != nil {
		return fmt.Errorf("prepare monitoring data: %w", err)
	}

	rollbackName := ContainerName + "-rollback"
	_, _ = client.Run(ctx, docker+" rm -f "+rollbackName+" >/dev/null 2>&1 || true")
	hadOld := containerExists(ctx, client, ContainerName)
	if hadOld {
		if _, err := client.Run(ctx, docker+" stop "+ContainerName+" && "+docker+" rename "+ContainerName+" "+rollbackName); err != nil {
			return fmt.Errorf("prepare monitoring rollback: %w", err)
		}
	}

	publishedAddress := net.JoinHostPort(cfg.PrivateAddress, strconv.Itoa(cfg.HostPort))
	run := fmt.Sprintf(
		"%s run -d --name %s --restart unless-stopped --network uploy -p %s:%d "+
			"-v /var/run/docker.sock:/var/run/docker.sock:ro -v %s:/data "+
			"-e UPLOY_MONITOR_CONTROL_TOKEN=%s -e UPLOY_MONITOR_READER_TOKEN=%s "+
			"-e UPLOY_MONITOR_RETENTION_DAYS=%d %s",
		docker, ContainerName, publishedAddress, AgentPort, dataDir,
		ssh.ShellQuote(cfg.ControlToken), ssh.ShellQuote(cfg.ReaderToken), cfg.RetentionDays, ssh.ShellQuote(cfg.Image),
	)
	if _, err := client.Run(ctx, run); err != nil {
		rollback(ctx, client, hadOld, rollbackName)
		return fmt.Errorf("start monitoring agent: %w", err)
	}
	if err := waitHealthy(ctx, client); err != nil {
		rollback(ctx, client, hadOld, rollbackName)
		return err
	}

	routeID := routeID(serverID)
	if cfg.FQDN == "" {
		if err := proxy.RemoveRoute(client, routeID); err != nil {
			rollback(ctx, client, hadOld, rollbackName)
			return fmt.Errorf("remove monitoring public route: %w", err)
		}
	} else if err := proxy.SetMonitoringRoute(client, routeID, []string{cfg.FQDN}, ContainerName, AgentPort); err != nil {
		rollback(ctx, client, hadOld, rollbackName)
		return fmt.Errorf("publish monitoring route: %w", err)
	}
	if hadOld {
		_, _ = client.Run(ctx, docker+" rm -f "+rollbackName+" >/dev/null 2>&1 || true")
	}
	return nil
}

func ValidateConfig(cfg Config) error {
	if !pinnedImage(cfg.Image) {
		return errors.New("monitoring image must use a pinned tag or digest")
	}
	if cfg.HostPort < 1 || cfg.HostPort > 65535 {
		return errors.New("monitoring port is invalid")
	}
	if !privateIP(cfg.PrivateAddress) {
		return errors.New("monitoring private address must be an RFC1918, ULA, or CGNAT IP")
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

func PrivateURL(address string, port int) string {
	return "http://" + net.JoinHostPort(address, strconv.Itoa(port))
}

func privateIP(value string) bool {
	ip := net.ParseIP(value)
	if ip == nil || ip.IsUnspecified() || ip.IsLoopback() || ip.IsMulticast() {
		return false
	}
	if ip.IsPrivate() {
		return true
	}
	ipv4 := ip.To4()
	return ipv4 != nil && ipv4[0] == 100 && ipv4[1] >= 64 && ipv4[1] <= 127
}

func GetHistories(ctx context.Context, baseURL, controlToken string, deploymentIDs []string, from, to time.Time, maxPoints int) (HistoriesResponse, error) {
	if len(deploymentIDs) == 0 {
		return HistoriesResponse{Deployments: map[string]HistoryResponse{}}, nil
	}
	body, err := json.Marshal(struct {
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/v1/history", strings.NewReader(string(body)))
	if err != nil {
		return HistoriesResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+controlToken)
	response, err := httpClient.Do(req)
	if err != nil {
		return HistoriesResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return HistoriesResponse{}, fmt.Errorf("monitoring agent HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	var histories HistoriesResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 16<<20)).Decode(&histories); err != nil {
		return HistoriesResponse{}, err
	}
	if histories.Deployments == nil {
		histories.Deployments = map[string]HistoryResponse{}
	}
	return histories, nil
}

func GetLatestAll(ctx context.Context, baseURL, controlToken string) (LatestResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/v1/latest", nil)
	if err != nil {
		return LatestResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+controlToken)
	response, err := liveHTTPClient.Do(req)
	if err != nil {
		return LatestResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return LatestResponse{}, fmt.Errorf("monitoring agent HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	var latest LatestResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 8<<20)).Decode(&latest); err != nil {
		return LatestResponse{}, err
	}
	if latest.Points == nil {
		latest.Points = []HistoryPoint{}
	}
	return latest, nil
}

func DeleteHistory(ctx context.Context, baseURL, controlToken string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, strings.TrimRight(baseURL, "/")+"/v1/history", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+controlToken)
	response, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("monitoring agent HTTP %d", response.StatusCode)
	}
	return nil
}

var httpClient = &http.Client{Timeout: 15 * time.Second}
var liveHTTPClient = &http.Client{Timeout: 5 * time.Second}

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
		status, err := client.Run(deadline, fmt.Sprintf("%s inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' %s", client.DockerBin(), ContainerName))
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

func rollback(ctx context.Context, client *ssh.Client, hadOld bool, rollbackName string) {
	docker := client.DockerBin()
	_, _ = client.Run(ctx, docker+" rm -f "+ContainerName+" >/dev/null 2>&1 || true")
	if hadOld {
		_, _ = client.Run(ctx, docker+" rename "+rollbackName+" "+ContainerName+" && "+docker+" start "+ContainerName)
	}
}

func containerExists(ctx context.Context, client *ssh.Client, name string) bool {
	_, err := client.Run(ctx, client.DockerBin()+" inspect "+name)
	return err == nil
}
