package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const (
	listenAddr       = ":9184"
	liveInterval     = 15 * time.Second
	persistInterval  = time.Minute
	defaultRetention = 7
	maxRetention     = 30
	maxHistoryPoints = 1000
	maxBatchPoints   = 10_000
	maxDatabaseBytes = 512 << 20
	minTokenLength   = 32
)

type config struct {
	controlToken string
	readerToken  string
	retention    int
	databasePath string
}

type sample struct {
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

type historyResponse struct {
	Points []sample `json:"points"`
}

type historyQuery struct {
	DeploymentIDs []string `json:"deployment_ids"`
	From          string   `json:"from"`
	To            string   `json:"to"`
	MaxPoints     int      `json:"max_points"`
}

type collector struct {
	db        *sql.DB
	retention time.Duration

	mu          sync.RWMutex
	latest      []sample
	lastPersist time.Time
	running     bool
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}
	db, err := openDatabase(cfg.databasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	collector := &collector{db: db, retention: time.Duration(cfg.retention) * 24 * time.Hour}
	collector.collect(context.Background(), true)
	go collector.run()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.Handle("GET /metrics", requireToken(cfg, http.HandlerFunc(collector.metricsHandler)))
	mux.Handle("GET /v1/latest", requireToken(cfg, http.HandlerFunc(collector.latestAllHandler)))
	mux.Handle("GET /v1/history", requireToken(cfg, http.HandlerFunc(collector.historiesHandler)))
	mux.Handle("POST /v1/history", requireToken(cfg, http.HandlerFunc(collector.historiesQueryHandler)))
	mux.Handle("GET /v1/deployments/{id}/latest", requireToken(cfg, http.HandlerFunc(collector.latestHandler)))
	mux.Handle("GET /v1/deployments/{id}/history", requireToken(cfg, http.HandlerFunc(collector.historyHandler)))
	mux.Handle("DELETE /v1/history", requireControlToken(cfg, http.HandlerFunc(collector.deleteHistoryHandler)))

	server := &http.Server{
		Addr:              envDefault("UPLOY_MONITOR_LISTEN_ADDR", listenAddr),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("uploy monitor listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func loadConfig() (config, error) {
	cfg := config{
		controlToken: os.Getenv("UPLOY_MONITOR_CONTROL_TOKEN"),
		readerToken:  os.Getenv("UPLOY_MONITOR_READER_TOKEN"),
		retention:    envInt("UPLOY_MONITOR_RETENTION_DAYS", defaultRetention),
		databasePath: envDefault("UPLOY_MONITOR_DB", "/data/metrics.db"),
	}
	if len(cfg.controlToken) < minTokenLength || len(cfg.readerToken) < minTokenLength {
		return config{}, fmt.Errorf("UPLOY_MONITOR_CONTROL_TOKEN and UPLOY_MONITOR_READER_TOKEN must each contain at least %d characters", minTokenLength)
	}
	if cfg.controlToken == cfg.readerToken {
		return config{}, errors.New("UPLOY_MONITOR_CONTROL_TOKEN and UPLOY_MONITOR_READER_TOKEN must differ")
	}
	if cfg.retention < 1 || cfg.retention > maxRetention {
		return config{}, fmt.Errorf("UPLOY_MONITOR_RETENTION_DAYS must be between 1 and %d", maxRetention)
	}
	return cfg, nil
}

func openDatabase(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	for _, statement := range []string{
		"PRAGMA auto_vacuum = INCREMENTAL",
		fmt.Sprintf("PRAGMA max_page_count = %d", maxDatabaseBytes/4096),
		`CREATE TABLE IF NOT EXISTS deployment_metrics (
			deployment_id TEXT NOT NULL,
			sampled_at INTEGER NOT NULL,
			container_id TEXT NOT NULL,
			container_name TEXT NOT NULL,
			state TEXT NOT NULL,
			cpu_percent REAL NOT NULL,
			memory_used_bytes INTEGER NOT NULL,
			memory_limit_bytes INTEGER NOT NULL,
			network_in_bytes_total INTEGER NOT NULL,
			network_out_bytes_total INTEGER NOT NULL,
			uptime_seconds INTEGER NOT NULL,
			PRIMARY KEY (deployment_id, sampled_at)
		)`,
		"CREATE INDEX IF NOT EXISTS idx_deployment_metrics_history ON deployment_metrics (deployment_id, sampled_at)",
	} {
		if _, err := db.Exec(statement); err != nil {
			db.Close()
			return nil, err
		}
	}
	return db, nil
}

func (c *collector) run() {
	ticker := time.NewTicker(liveInterval)
	defer ticker.Stop()
	for range ticker.C {
		c.collect(context.Background(), false)
	}
}

func (c *collector) collect(ctx context.Context, forcePersist bool) {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	persist := forcePersist || time.Since(c.lastPersist) >= persistInterval
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()
	}()

	samples, err := readDockerSamples(ctx)
	if err != nil {
		log.Printf("collect metrics: %v", err)
		return
	}
	c.mu.Lock()
	c.latest = samples
	c.mu.Unlock()
	if !persist {
		return
	}
	if err := c.persist(samples); err != nil {
		log.Printf("persist metrics: %v", err)
		return
	}
	c.mu.Lock()
	c.lastPersist = time.Now()
	c.mu.Unlock()
}

func (c *collector) persist(samples []sample) error {
	if err := c.deleteExpired(); err != nil {
		return err
	}
	if len(samples) == 0 {
		return nil
	}
	write := func() error {
		tx, err := c.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		statement, err := tx.Prepare(`INSERT OR REPLACE INTO deployment_metrics (
			deployment_id, sampled_at, container_id, container_name, state, cpu_percent, memory_used_bytes,
			memory_limit_bytes, network_in_bytes_total, network_out_bytes_total, uptime_seconds
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		defer statement.Close()
		for _, metric := range samples {
			if _, err := statement.Exec(metric.DeploymentID, metric.SampledAt, metric.ContainerID, metric.ContainerName, metric.State,
				metric.CPUPercent, metric.MemoryUsedBytes, metric.MemoryLimitBytes, metric.NetworkInBytesTotal,
				metric.NetworkOutBytesTotal, metric.UptimeSeconds); err != nil {
				return err
			}
		}
		return tx.Commit()
	}
	if err := write(); err == nil {
		return nil
	} else if !strings.Contains(strings.ToLower(err.Error()), "full") {
		return err
	}
	if err := c.pruneOldest(); err != nil {
		return err
	}
	return write()
}

func (c *collector) deleteExpired() error {
	_, err := c.db.Exec("DELETE FROM deployment_metrics WHERE sampled_at < ?", time.Now().Add(-c.retention).UnixMilli())
	return err
}

func (c *collector) pruneOldest() error {
	_, err := c.db.Exec(`DELETE FROM deployment_metrics WHERE rowid IN (
		SELECT rowid FROM deployment_metrics ORDER BY sampled_at ASC LIMIT (
			SELECT MAX(1, COUNT(*) / 10) FROM deployment_metrics
		)
	)`)
	if err != nil {
		return err
	}
	_, _ = c.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	_, _ = c.db.Exec("PRAGMA incremental_vacuum")
	return nil
}

func readDockerSamples(ctx context.Context) ([]sample, error) {
	psOutput, err := dockerOutput(ctx, "ps", "-a", "--no-trunc", "--filter", "label=uploy.deployment_id", "--format", "{{.ID}}|{{.Names}}|{{.Label \"uploy.deployment_id\"}}|{{.State}}")
	if err != nil {
		return nil, err
	}
	if psOutput == "" {
		return []sample{}, nil
	}
	byID := make(map[string]sample)
	var ids []string
	for _, line := range strings.Split(psOutput, "\n") {
		parts := strings.Split(line, "|")
		if len(parts) != 4 || parts[0] == "" || parts[2] == "" {
			continue
		}
		byID[parts[0]] = sample{
			DeploymentID:  parts[2],
			ContainerID:   parts[0],
			ContainerName: parts[1],
			State:         parts[3],
			SampledAt:     time.Now().UTC().UnixMilli(),
		}
		if parts[3] == "running" {
			ids = append(ids, parts[0])
		}
	}
	if len(byID) == 0 {
		return []sample{}, nil
	}
	if len(ids) == 0 {
		return sortedSamples(byID), nil
	}
	statsOutput, err := dockerOutput(ctx, append([]string{"stats", "--no-stream", "--no-trunc", "--format", "{{.ID}}|{{.CPUPerc}}|{{.MemUsage}}|{{.NetIO}}"}, ids...)...)
	if err != nil {
		return nil, err
	}
	startedOutput, err := dockerOutput(ctx, append([]string{"inspect", "--format", "{{.Id}}|{{.State.StartedAt}}"}, ids...)...)
	if err != nil {
		return nil, err
	}
	startedAt := make(map[string]time.Time)
	for _, line := range strings.Split(startedOutput, "\n") {
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		if parsed, err := time.Parse(time.RFC3339Nano, parts[1]); err == nil {
			startedAt[parts[0]] = parsed
		}
	}
	now := time.Now().UTC()
	for _, line := range strings.Split(statsOutput, "\n") {
		parts := strings.Split(line, "|")
		if len(parts) != 4 {
			continue
		}
		metric, found := byID[parts[0]]
		if !found {
			continue
		}
		cpu, err := parsePercent(parts[1])
		if err != nil {
			continue
		}
		used, limit, err := parsePair(parts[2])
		if err != nil {
			continue
		}
		networkIn, networkOut, err := parsePair(parts[3])
		if err != nil {
			continue
		}
		metric.SampledAt = now.UnixMilli()
		metric.CPUPercent = cpu
		metric.MemoryUsedBytes = used
		metric.MemoryLimitBytes = limit
		metric.NetworkInBytesTotal = networkIn
		metric.NetworkOutBytesTotal = networkOut
		if started, ok := startedAt[metric.ContainerID]; ok {
			metric.UptimeSeconds = max(0, int64(now.Sub(started).Seconds()))
		}
		byID[metric.ContainerID] = metric
	}
	return sortedSamples(byID), nil
}

func sortedSamples(byID map[string]sample) []sample {
	metrics := make([]sample, 0, len(byID))
	for _, metric := range byID {
		metrics = append(metrics, metric)
	}
	sort.Slice(metrics, func(left, right int) bool {
		return metrics[left].DeploymentID < metrics[right].DeploymentID
	})
	return metrics
}

func dockerOutput(ctx context.Context, args ...string) (string, error) {
	output, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker %s: %w: %s", strings.Join(args[:min(len(args), 2)], " "), err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func parsePercent(value string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(value), "%"), 64)
}

func parsePair(value string) (int64, int64, error) {
	parts := strings.Split(value, " / ")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid metric pair %q", value)
	}
	first, err := parseBytes(parts[0])
	if err != nil {
		return 0, 0, err
	}
	second, err := parseBytes(parts[1])
	return first, second, err
}

func parseBytes(value string) (int64, error) {
	value = strings.TrimSpace(value)
	multipliers := []struct {
		suffix string
		value  float64
	}{
		{"TiB", 1 << 40}, {"GiB", 1 << 30}, {"MiB", 1 << 20}, {"KiB", 1 << 10},
		{"TB", 1e12}, {"GB", 1e9}, {"MB", 1e6}, {"kB", 1e3}, {"B", 1},
	}
	for _, unit := range multipliers {
		if strings.HasSuffix(value, unit.suffix) {
			number := strings.TrimSpace(strings.TrimSuffix(value, unit.suffix))
			parsed, err := strconv.ParseFloat(number, 64)
			if err != nil {
				return 0, err
			}
			return int64(math.Round(parsed * unit.value)), nil
		}
	}
	return 0, fmt.Errorf("unknown byte unit %q", value)
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte("{\"status\":\"ok\"}\n"))
}

func requireToken(cfg config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" || !validToken(token, cfg.controlToken, cfg.readerToken) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="uploy-monitor"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requireControlToken(cfg config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" || !sameToken(token, cfg.controlToken) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="uploy-monitor"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func validToken(token, control, reader string) bool {
	return sameToken(token, control) || sameToken(token, reader)
}

func sameToken(left, right string) bool {
	leftHash := sha256.Sum256([]byte(left))
	rightHash := sha256.Sum256([]byte(right))
	return subtle.ConstantTimeCompare(leftHash[:], rightHash[:]) == 1
}

func (c *collector) metricsHandler(w http.ResponseWriter, _ *http.Request) {
	c.mu.RLock()
	metrics := append([]sample(nil), c.latest...)
	c.mu.RUnlock()
	w.Header().Set("Content-Type", "application/openmetrics-text; version=1.0.0; charset=utf-8")
	for _, metric := range metrics {
		labels := fmt.Sprintf(`deployment_id=%q,container_id=%q,container_name=%q`, metric.DeploymentID, metric.ContainerID, metric.ContainerName)
		fmt.Fprintf(w, "uploy_container_cpu_percent{%s} %g\n", labels, metric.CPUPercent)
		fmt.Fprintf(w, "uploy_container_running{%s} %d\n", labels, boolInt(metric.State == "running"))
		fmt.Fprintf(w, "uploy_container_memory_used_bytes{%s} %d\n", labels, metric.MemoryUsedBytes)
		fmt.Fprintf(w, "uploy_container_memory_limit_bytes{%s} %d\n", labels, metric.MemoryLimitBytes)
		fmt.Fprintf(w, "uploy_container_network_receive_bytes_total{%s} %d\n", labels, metric.NetworkInBytesTotal)
		fmt.Fprintf(w, "uploy_container_network_transmit_bytes_total{%s} %d\n", labels, metric.NetworkOutBytesTotal)
		fmt.Fprintf(w, "uploy_container_uptime_seconds{%s} %d\n", labels, metric.UptimeSeconds)
		fmt.Fprintf(w, "uploy_container_sample_timestamp_seconds{%s} %d\n", labels, metric.SampledAt/1000)
	}
	_, _ = w.Write([]byte("# EOF\n"))
}

func (c *collector) latestHandler(w http.ResponseWriter, r *http.Request) {
	deploymentID := r.PathValue("id")
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, metric := range c.latest {
		if metric.DeploymentID == deploymentID {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(metric); err != nil {
				log.Printf("write latest: %v", err)
			}
			return
		}
	}
	http.Error(w, "deployment not found", http.StatusNotFound)
}

func (c *collector) latestAllHandler(w http.ResponseWriter, _ *http.Request) {
	c.mu.RLock()
	metrics := append([]sample(nil), c.latest...)
	c.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"points":`))
	if err := writeJSON(w, metrics); err != nil {
		log.Printf("write latest: %v", err)
	}
	_, _ = w.Write([]byte("}\n"))
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func (c *collector) historyHandler(w http.ResponseWriter, r *http.Request) {
	deploymentID := r.PathValue("id")
	start, end, maxPoints, ok := parseHistoryQuery(w, r)
	if !ok || !validDeploymentID(deploymentID) {
		if !ok {
			return
		}
		http.Error(w, "invalid deployment id", http.StatusBadRequest)
		return
	}
	metrics, err := c.queryHistory(deploymentID, start.UnixMilli(), end.UnixMilli(), maxPoints)
	if err != nil {
		log.Printf("query history deployment=%s: %v", deploymentID, err)
		http.Error(w, "failed to read history", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"points":`))
	if err := writeJSON(w, metrics); err != nil {
		log.Printf("write history: %v", err)
	}
	_, _ = w.Write([]byte("}\n"))
}

func (c *collector) historiesHandler(w http.ResponseWriter, r *http.Request) {
	start, end, maxPoints, ok := parseHistoryQuery(w, r)
	if !ok {
		return
	}
	ids := strings.Split(r.URL.Query().Get("deployment_ids"), ",")
	c.writeHistories(w, ids, start, end, maxPoints)
}

func (c *collector) historiesQueryHandler(w http.ResponseWriter, r *http.Request) {
	var query historyQuery
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&query); err != nil {
		http.Error(w, "invalid history query", http.StatusBadRequest)
		return
	}
	start, end, maxPoints, ok := parseHistoryValues(w, query.From, query.To, query.MaxPoints)
	if !ok {
		return
	}
	c.writeHistories(w, query.DeploymentIDs, start, end, maxPoints)
}

func (c *collector) writeHistories(w http.ResponseWriter, ids []string, start, end time.Time, maxPoints int) {
	if len(ids) == 0 || len(ids) > 500 {
		http.Error(w, "deployment_ids must contain 1 to 500 ids", http.StatusBadRequest)
		return
	}
	if maxPoints > maxBatchPoints/len(ids) {
		http.Error(w, fmt.Sprintf("requested history exceeds %d total points", maxBatchPoints), http.StatusBadRequest)
		return
	}
	histories := make(map[string]historyResponse, len(ids))
	for _, deploymentID := range ids {
		if !validDeploymentID(deploymentID) {
			http.Error(w, "invalid deployment id", http.StatusBadRequest)
			return
		}
		metrics, err := c.queryHistory(deploymentID, start.UnixMilli(), end.UnixMilli(), maxPoints)
		if err != nil {
			log.Printf("query history deployment=%s: %v", deploymentID, err)
			http.Error(w, "failed to read history", http.StatusInternalServerError)
			return
		}
		histories[deploymentID] = historyResponse{Points: metrics}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		Deployments map[string]historyResponse `json:"deployments"`
	}{Deployments: histories}); err != nil {
		log.Printf("write histories: %v", err)
	}
}

func parseHistoryQuery(w http.ResponseWriter, r *http.Request) (time.Time, time.Time, int, bool) {
	return parseHistoryValues(w, r.URL.Query().Get("from"), r.URL.Query().Get("to"), envIntValue(r.URL.Query().Get("max_points"), maxHistoryPoints))
}

func parseHistoryValues(w http.ResponseWriter, rawFrom, rawTo string, maxPoints int) (time.Time, time.Time, int, bool) {
	end := time.Now().UTC()
	start := end.Add(-7 * 24 * time.Hour)
	var err error
	if rawFrom != "" {
		start, err = time.Parse(time.RFC3339, rawFrom)
		if err != nil {
			http.Error(w, "invalid from", http.StatusBadRequest)
			return time.Time{}, time.Time{}, 0, false
		}
	}
	if rawTo != "" {
		end, err = time.Parse(time.RFC3339, rawTo)
		if err != nil {
			http.Error(w, "invalid to", http.StatusBadRequest)
			return time.Time{}, time.Time{}, 0, false
		}
	}
	if !end.After(start) || end.Sub(start) > time.Duration(maxRetention)*24*time.Hour {
		http.Error(w, "range must be positive and at most 30 days", http.StatusBadRequest)
		return time.Time{}, time.Time{}, 0, false
	}
	if maxPoints < 1 || maxPoints > maxHistoryPoints {
		http.Error(w, "max_points must be between 1 and 1000", http.StatusBadRequest)
		return time.Time{}, time.Time{}, 0, false
	}
	return start, end, maxPoints, true
}

func validDeploymentID(value string) bool {
	return value != "" && len(value) <= 200 && !strings.ContainsAny(value, " \t\r\n")
}

func (c *collector) deleteHistoryHandler(w http.ResponseWriter, _ *http.Request) {
	if _, err := c.db.Exec("DELETE FROM deployment_metrics"); err != nil {
		log.Printf("delete history: %v", err)
		http.Error(w, "failed to delete history", http.StatusInternalServerError)
		return
	}
	_, _ = c.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	_, _ = c.db.Exec("VACUUM")
	w.WriteHeader(http.StatusNoContent)
}

func (c *collector) queryHistory(deploymentID string, from, to int64, maxPoints int) ([]sample, error) {
	rows, err := c.db.Query(`SELECT deployment_id, container_id, container_name, state, sampled_at, cpu_percent,
		memory_used_bytes, memory_limit_bytes, network_in_bytes_total, network_out_bytes_total, uptime_seconds
		FROM deployment_metrics WHERE deployment_id = ? AND sampled_at >= ? AND sampled_at <= ? ORDER BY sampled_at ASC`, deploymentID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	raw := []sample{}
	for rows.Next() {
		var metric sample
		if err := rows.Scan(&metric.DeploymentID, &metric.ContainerID, &metric.ContainerName, &metric.State, &metric.SampledAt, &metric.CPUPercent,
			&metric.MemoryUsedBytes, &metric.MemoryLimitBytes, &metric.NetworkInBytesTotal, &metric.NetworkOutBytesTotal, &metric.UptimeSeconds); err != nil {
			return nil, err
		}
		raw = append(raw, metric)
	}
	if err := rows.Err(); err != nil || len(raw) <= maxPoints {
		return raw, err
	}
	return downsample(raw, from, to, maxPoints), nil
}

func downsample(raw []sample, from, to int64, maxPoints int) []sample {
	span := max(int64(1), to-from+1)
	width := max(int64(1), (span+int64(maxPoints)-1)/int64(maxPoints))
	result := make([]sample, 0, maxPoints)
	for _, metric := range raw {
		bucket := (metric.SampledAt - from) / width
		if len(result) > 0 && (result[len(result)-1].SampledAt-from)/width == bucket {
			result[len(result)-1] = metric
			continue
		}
		result = append(result, metric)
	}
	return result
}

func writeJSON(w http.ResponseWriter, value any) error {
	return json.NewEncoder(w).Encode(value)
}

func envDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	return envIntValue(os.Getenv(key), fallback)
}

func envIntValue(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func max(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
