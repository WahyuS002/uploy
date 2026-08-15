package jobs

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/WahyuS002/uploy/broker"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/proxy"
	"github.com/WahyuS002/uploy/ssh"
	"github.com/jackc/pgx/v5"
)

const proxyContainerName = "uploy-proxy"

// TLS foreground polling: check immediately, then retry up to
// tlsForegroundAttempts−1 more times with tlsRetryInterval pauses
// (~50 s total) before deferring to the background reconciler.
const (
	tlsForegroundAttempts = 6
	tlsRetryInterval      = 10 * time.Second
	healthPollInterval    = 2 * time.Second
	drainPeriod           = 30 * time.Second
)

type DeployConfig struct {
	DeploymentID       string
	ServiceID          string
	Image              string
	ContainerName      string
	HealthcheckCommand string
	// ContainerPort is what the image listens on inside the container; HostPort
	// is where it is published on the machine. They are equal for a database
	// reached on its own well-known number, and different for anything
	// listening on 80.
	ContainerPort int
	HostPort      int
	EnvVars       []db.EnvPair
	Domains       []string
	ServerID      string
	Server        ssh.ServerConfig
}

func appendLog(ctx context.Context, deploymentID, msg, logType, phase string) {
	if err := db.AppendLog(ctx, deploymentID, msg, logType, phase); err != nil {
		log.Printf("AppendLog deploymentID=%s error: %v", deploymentID, err)
	}
}

func failDeploy(deploymentID, msg string) {
	log.Println(msg)

	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	appendLog(cleanupCtx, deploymentID, msg, "stderr", "")
	if err := db.SetDeploymentStatus(cleanupCtx, deploymentID, "failed"); err != nil {
		log.Printf("SetDeploymentStatus deploymentID=%s error: %v", deploymentID, err)
		return
	}

	appendLog(cleanupCtx, deploymentID, "deployment failed", "stderr", "failed")
	broker.PublishDone(deploymentID, "failed")
}

func finishDeploy(deploymentID, status string) {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.SetDeploymentStatus(cleanupCtx, deploymentID, status); err != nil {
		log.Printf("SetDeploymentStatus deploymentID=%s error: %v", deploymentID, err)
		return
	}

	appendLog(cleanupCtx, deploymentID, fmt.Sprintf("deployment %s", status), "stdout", "complete")
	broker.PublishDone(deploymentID, status)
}

func RunDeploy(cfg DeployConfig) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered deploymentID=%s: %v\n%s", cfg.DeploymentID, r, debug.Stack())
			failDeploy(cfg.DeploymentID, fmt.Sprintf("panic: %v", r))
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	hasDomains := len(cfg.Domains) > 0

	appendLog(ctx, cfg.DeploymentID, "connecting to server...", "stdout", "connect")

	client, err := ssh.NewClient(cfg.Server)
	if err != nil {
		failDeploy(cfg.DeploymentID, "SSH connection failed: "+err.Error())
		return
	}
	defer client.Close()

	if err := client.DetectDocker(); err != nil {
		failDeploy(cfg.DeploymentID, err.Error())
		return
	}

	docker := client.DockerBin()

	// step 1: docker pull
	appendLog(ctx, cfg.DeploymentID, "pulling image...", "stdout", "pull_image")
	if !runStep(ctx, client, cfg.DeploymentID, docker+" pull "+cfg.Image) {
		return
	}

	// Every container joins the uploy network, so it has to exist before any of
	// them starts — not only when the proxy is being set up.
	if err := proxy.EnsureNetwork(client); err != nil {
		failDeploy(cfg.DeploymentID, "Network setup failed: "+err.Error())
		return
	}

	oldDeployment, oldConfig, hasOld, err := previousDeployment(ctx, client, cfg.ServiceID, cfg.ContainerName)
	if err != nil {
		failDeploy(cfg.DeploymentID, "Could not resolve active deployment: "+err.Error())
		return
	}

	if hasDomains {
		if !runRollingDeploy(ctx, client, docker, cfg, oldDeployment, oldConfig, hasOld) {
			return
		}
		reconcileDomainCertificates(ctx, client, cfg)
		finishDeploy(cfg.DeploymentID, "success")
		return
	}

	oldContainer := cfg.ContainerName
	if hasOld {
		oldContainer = ContainerNameForDeployment(oldConfig, oldDeployment.ID)
		if len(oldConfig.Domains) > 0 {
			appendLog(ctx, cfg.DeploymentID, "removing previous domain route...", "stdout", "proxy_setup")
			if err := proxy.RemoveRoute(client, cfg.ServiceID); err != nil {
				failDeploy(cfg.DeploymentID, "Could not remove previous domain route: "+err.Error())
				return
			}
		}
	}

	appendLog(ctx, cfg.DeploymentID, "stopping existing container...", "stdout", "stop_container")
	if !stopAndRemoveContainer(ctx, client, cfg.DeploymentID, oldContainer) {
		return
	}

	appendLog(ctx, cfg.DeploymentID, "starting application container...", "stdout", "start_container")
	if !runStep(ctx, client, cfg.DeploymentID, buildDockerRunCmd(docker, cfg)) {
		return
	}

	finishDeploy(cfg.DeploymentID, "success")
}

func runRollingDeploy(ctx context.Context, client *ssh.Client, docker string, cfg DeployConfig, oldDeployment db.Deployment, oldConfig db.ServiceConfig, hasOld bool) bool {
	appendLog(ctx, cfg.DeploymentID, "checking rolling proxy capability...", "stdout", "proxy_setup")
	if err := proxy.VerifyRollingReady(ctx, client); err != nil {
		if dbErr := db.SetServerProxyError(ctx, cfg.ServerID, "degraded", err.Error()); dbErr != nil {
			log.Printf("SetServerProxyError serverID=%s error: %v", cfg.ServerID, dbErr)
		}
		failDeploy(cfg.DeploymentID, "Rolling proxy is not ready; upgrade it from the Servers page: "+err.Error())
		return false
	}

	appendLog(ctx, cfg.DeploymentID, "checking proxy ports...", "stdout", "proxy_setup")
	oldContainer := cfg.ContainerName
	if hasOld {
		oldContainer = ContainerNameForDeployment(oldConfig, oldDeployment.ID)
	}
	releaseCurrent, err := checkProxyPortConflicts(client, oldContainer)
	if err != nil {
		failDeploy(cfg.DeploymentID, "Proxy setup failed: "+err.Error())
		return false
	}
	if releaseCurrent {
		failDeploy(cfg.DeploymentID, "Rolling deployment is unavailable while the current container owns ports 80 or 443")
		return false
	}

	if hasOld && len(oldConfig.Domains) > 0 {
		if err := proxy.SetRoute(client, cfg.ServiceID, oldConfig.Domains, oldContainer, int(oldConfig.ContainerPort)); err != nil {
			failDeploy(cfg.DeploymentID, "Could not migrate the active route: "+err.Error())
			return false
		}
		if err := proxy.WaitForRoute(ctx, client, cfg.ServiceID, oldContainer, int(oldConfig.ContainerPort)); err != nil {
			failDeploy(cfg.DeploymentID, "Could not confirm the active route migration: "+err.Error())
			return false
		}
	}

	hasHealthcheck, err := imageHasHealthcheck(ctx, client, docker, cfg.Image)
	if err != nil {
		failDeploy(cfg.DeploymentID, "Could not inspect image healthcheck: "+err.Error())
		return false
	}

	candidateName := DeploymentContainerName(cfg.ContainerName, cfg.DeploymentID)
	candidate := cfg
	candidate.ContainerName = candidateName
	if !hasHealthcheck {
		candidate.HealthcheckCommand = fallbackHealthcheckCommand(cfg.ContainerPort)
		appendLog(ctx, cfg.DeploymentID, fmt.Sprintf("image has no Docker healthcheck; using HTTP readiness check on port %d...", cfg.ContainerPort), "stdout", "health_check")
	}
	appendLog(ctx, cfg.DeploymentID, "starting candidate container...", "stdout", "start_container")
	if !runStep(ctx, client, cfg.DeploymentID, buildDockerRunCmd(docker, candidate)) {
		return false
	}

	appendLog(ctx, cfg.DeploymentID, "waiting for candidate healthcheck...", "stdout", "health_check")
	if err := waitForHealthy(ctx, client, docker, candidateName); err != nil {
		_ = stopAndRemoveContainer(ctx, client, cfg.DeploymentID, candidateName)
		failDeploy(cfg.DeploymentID, "Candidate did not become healthy: "+err.Error())
		return false
	}

	appendLog(ctx, cfg.DeploymentID, "switching traffic to candidate...", "stdout", "cutover")
	if err := proxy.SetRoute(client, cfg.ServiceID, cfg.Domains, candidateName, cfg.ContainerPort); err != nil {
		_ = stopAndRemoveContainer(ctx, client, cfg.DeploymentID, candidateName)
		failDeploy(cfg.DeploymentID, "Traffic cutover failed: "+err.Error())
		return false
	}
	if err := proxy.WaitForRoute(ctx, client, cfg.ServiceID, candidateName, cfg.ContainerPort); err != nil {
		restoreErr := restorePreviousRoute(ctx, client, cfg.ServiceID, oldConfig, oldContainer, hasOld)
		if restoreErr == nil {
			_ = stopAndRemoveContainer(ctx, client, cfg.DeploymentID, candidateName)
			failDeploy(cfg.DeploymentID, "Traffic cutover was not confirmed; previous route restored: "+err.Error())
		} else {
			failDeploy(cfg.DeploymentID, "Traffic cutover was not confirmed and rollback could not be confirmed; candidate left running: "+errors.Join(err, restoreErr).Error())
		}
		return false
	}

	if !hasOld {
		appendLog(ctx, cfg.DeploymentID, "traffic is live", "stdout", "active")
		return true
	}

	appendLog(ctx, cfg.DeploymentID, "draining previous container for 30 seconds...", "stdout", "drain")
	if err := monitorHealthy(ctx, client, docker, candidateName, drainPeriod); err != nil {
		if restoreErr := restorePreviousRoute(ctx, client, cfg.ServiceID, oldConfig, oldContainer, true); restoreErr != nil {
			failDeploy(cfg.DeploymentID, "Candidate failed during drain and rollback could not be confirmed; both containers left running: "+errors.Join(err, restoreErr).Error())
			return false
		}
		_ = stopAndRemoveContainer(ctx, client, cfg.DeploymentID, candidateName)
		failDeploy(cfg.DeploymentID, "Candidate failed during drain; traffic restored to previous deployment: "+err.Error())
		return false
	}

	appendLog(ctx, cfg.DeploymentID, "removing drained container...", "stdout", "drain")
	return stopAndRemoveContainer(ctx, client, cfg.DeploymentID, oldContainer)
}

func previousDeployment(ctx context.Context, client *ssh.Client, serviceID, containerName string) (db.Deployment, db.ServiceConfig, bool, error) {
	deployment, cfg, err := db.GetLatestSuccessfulDeploymentConfig(ctx, serviceID)
	if err == nil {
		return deployment, cfg, true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Deployment{}, db.ServiceConfig{}, false, nil
	}
	if !errors.Is(err, db.ErrDeploymentSnapshotMissing) {
		return db.Deployment{}, db.ServiceConfig{}, false, err
	}

	legacy, legacyErr := proxy.LegacyRouteForContainer(ctx, client, containerName)
	if errors.Is(legacyErr, proxy.ErrLegacyRouteNotFound) {
		return deployment, db.ServiceConfig{}, false, nil
	}
	if legacyErr != nil {
		return db.Deployment{}, db.ServiceConfig{}, false, legacyErr
	}
	return deployment, db.ServiceConfig{
		SchemaVersion: 1,
		ContainerName: legacy.ContainerName,
		ContainerPort: int32(legacy.ContainerPort),
		Domains:       legacy.Domains,
	}, true, nil
}

func restorePreviousRoute(ctx context.Context, client *ssh.Client, serviceID string, oldConfig db.ServiceConfig, oldContainer string, hasOld bool) error {
	if hasOld && len(oldConfig.Domains) > 0 {
		if err := proxy.SetRoute(client, serviceID, oldConfig.Domains, oldContainer, int(oldConfig.ContainerPort)); err != nil {
			return err
		}
		return proxy.WaitForRoute(ctx, client, serviceID, oldContainer, int(oldConfig.ContainerPort))
	}
	if err := proxy.RemoveRoute(client, serviceID); err != nil {
		return err
	}
	return proxy.WaitForRouteRemoved(ctx, client, serviceID)
}

func DeploymentContainerName(base, deploymentID string) string {
	shortID := deploymentID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	return base + "-" + shortID
}

func ContainerNameForDeployment(cfg db.ServiceConfig, deploymentID string) string {
	if cfg.SchemaVersion >= 2 && len(cfg.Domains) > 0 {
		return DeploymentContainerName(cfg.ContainerName, deploymentID)
	}
	return cfg.ContainerName
}

func imageHasHealthcheck(ctx context.Context, client *ssh.Client, docker, image string) (bool, error) {
	output, err := client.Run(ctx, fmt.Sprintf("%s image inspect --format '{{if .Config.Healthcheck}}{{json .Config.Healthcheck.Test}}{{end}}' %s", docker, image))
	if err != nil {
		return false, err
	}
	return output != "" && output != "null" && output != `["NONE"]`, nil
}

func fallbackHealthcheckCommand(port int) string {
	return fmt.Sprintf("curl -fsS http://127.0.0.1:%d/ >/dev/null || wget -q -O /dev/null http://127.0.0.1:%d/ || exit 1", port, port)
}

func containerHealth(ctx context.Context, client *ssh.Client, docker, containerName string) (string, error) {
	return client.Run(ctx, fmt.Sprintf("%s inspect --format '{{.State.Health.Status}}' %s", docker, containerName))
}

func waitForHealthy(ctx context.Context, client *ssh.Client, docker, containerName string) error {
	for {
		status, err := containerHealth(ctx, client, docker, containerName)
		if err != nil {
			return err
		}
		switch status {
		case "healthy":
			return nil
		case "unhealthy":
			return fmt.Errorf("container is unhealthy")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(healthPollInterval):
		}
	}
}

func monitorHealthy(ctx context.Context, client *ssh.Client, docker, containerName string, duration time.Duration) error {
	deadline := time.NewTimer(duration)
	defer deadline.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return nil
		case <-time.After(healthPollInterval):
			status, err := containerHealth(ctx, client, docker, containerName)
			if err != nil {
				return err
			}
			if status != "healthy" {
				return fmt.Errorf("container health is %s", status)
			}
		}
	}
}

func reconcileDomainCertificates(ctx context.Context, client *ssh.Client, cfg DeployConfig) {
	if len(cfg.Domains) == 0 {
		return
	}

	if err := db.SetServerProxyReady(ctx, cfg.ServerID, "ready"); err != nil {
		log.Printf("SetServerProxyReady error: %v", err)
	}
	domainList, err := db.ListDomainsByService(ctx, cfg.ServiceID)
	if err != nil {
		log.Printf("ListDomainsByService error: %v", err)
		return
	}
	unresolvedDomains := make(map[string]string)
	for _, domain := range domainList {
		if domain.Status != "ready" {
			unresolvedDomains[domain.Domain] = domain.ID
		}
	}
	if len(unresolvedDomains) == 0 {
		return
	}
	appendLog(ctx, cfg.DeploymentID, "checking HTTPS certificate status...", "stdout", "tls_cert")
	for attempt := 0; attempt < tlsForegroundAttempts && len(unresolvedDomains) > 0; attempt++ {
		if attempt > 0 {
			time.Sleep(tlsRetryInterval)
		}
		for domain, domainID := range unresolvedDomains {
			ready, promoteErr := promoteDomainIfCertificateReady(ctx, client, cfg.Server.Host, domainID, domain)
			if ready && promoteErr == nil {
				appendLog(ctx, cfg.DeploymentID, fmt.Sprintf("HTTPS is ready for %s", domain), "stdout", "tls_cert")
				delete(unresolvedDomains, domain)
			}
		}
	}
}

func buildDockerRunCmd(docker string, cfg DeployConfig) string {
	var args string

	// Every container joins the shared network, published or not. It is how one
	// service reaches another by name, and for a service with no host port it is
	// the only way it can be reached at all.
	args = fmt.Sprintf("%s run -d --name %s --network uploy", docker, cfg.ContainerName)
	if cfg.ServiceID != "" {
		args += fmt.Sprintf(" --label uploy.service_id=%s", cfg.ServiceID)
	}
	if cfg.DeploymentID != "" {
		args += fmt.Sprintf(" --label uploy.deployment_id=%s", cfg.DeploymentID)
	}

	if len(cfg.Domains) > 0 {
		// Proxy mode: Traefik forwards to the container's internal port, so
		// nothing is published on the host.

		// Domain routing is owned by Traefik's watched file provider. Containers
		// deliberately carry no router labels, so a cutover can replace one backend
		// atomically without duplicate Docker-provider routers.
	} else if cfg.HostPort > 0 {
		// Publish on the chosen host port. Host and container port are two
		// different numbers whenever the image listens on 80 — publishing
		// port:port made nginx unreachable, because nothing inside the container
		// was listening on the number it was published as.
		args += fmt.Sprintf(" -p %d:%d", cfg.HostPort, cfg.ContainerPort)
	}
	// No domains and no host port means internal only: reachable by other
	// services on the uploy network, and by nothing outside the machine. That is
	// what a database should be unless someone asks otherwise.

	// Survive server reboots and daemon restarts. Without this a reboot silently
	// kills every deployed service until someone redeploys by hand.
	args += " --restart unless-stopped"
	if cfg.HealthcheckCommand != "" {
		args += fmt.Sprintf(" --health-cmd %s --health-interval 5s --health-timeout 5s --health-retries 10 --health-start-period 5s", ssh.ShellQuote(cfg.HealthcheckCommand))
	}

	for _, env := range cfg.EnvVars {
		args += fmt.Sprintf(" --env %s", ssh.ShellQuote(env.Key+"="+env.Value))
	}

	args += " " + cfg.Image
	return args
}

func RemoveService(server ssh.ServerConfig, serviceID, containerName string) error {
	client, err := ssh.NewClient(server)
	if err != nil {
		return fmt.Errorf("connect to server: %w", err)
	}
	defer client.Close()
	if err := client.DetectDocker(); err != nil {
		return err
	}
	if err := proxy.RemoveRoute(client, serviceID); err != nil {
		return fmt.Errorf("remove route: %w", err)
	}
	cmd := fmt.Sprintf("%s rm -f %s 2>/dev/null || true", client.DockerBin(), containerName)
	if _, err := captureStdoutLines(client, cmd); err != nil {
		return fmt.Errorf("remove container %s: %w", containerName, err)
	}
	return nil
}

func runStep(ctx context.Context, client *ssh.Client, deploymentID, command string) bool {
	stdoutCh, stderrCh, done := client.StreamCommand(command)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for line := range stdoutCh {
			appendLog(ctx, deploymentID, line, "stdout", "")
		}
	}()
	go func() {
		defer wg.Done()
		for line := range stderrCh {
			appendLog(ctx, deploymentID, line, "stderr", "")
		}
	}()

	wg.Wait()

	if err := <-done; err != nil {
		failDeploy(deploymentID, fmt.Sprintf("command failed: %v", err))
		return false
	}
	return true
}

func stopAndRemoveContainer(ctx context.Context, client *ssh.Client, deploymentID, containerName string) bool {
	docker := client.DockerBin()

	stopCmd := fmt.Sprintf("%s stop %s 2>/dev/null || true", docker, containerName)
	if !runStep(ctx, client, deploymentID, stopCmd) {
		return false
	}

	rmCmd := fmt.Sprintf("%s rm %s 2>/dev/null || true", docker, containerName)
	if !runStep(ctx, client, deploymentID, rmCmd) {
		return false
	}

	return true
}

func checkProxyPortConflicts(client *ssh.Client, currentContainer string) (bool, error) {
	releaseCurrent := false

	for _, port := range []int{80, 443} {
		owner, err := publishedPortOwner(client, port)
		if err != nil {
			return false, fmt.Errorf("check port %d owner: %w", port, err)
		}

		busy, err := isHostPortBusy(client, port)
		if err != nil {
			return false, fmt.Errorf("check port %d usage: %w", port, err)
		}

		switch {
		case owner == currentContainer:
			releaseCurrent = true
		case owner != "" && owner != proxyContainerName:
			return false, fmt.Errorf("port %d is already used by container %s; Traefik needs exclusive access to ports 80 and 443", port, owner)
		case owner == "" && busy:
			return false, fmt.Errorf("port %d is already in use by a non-Docker process; Traefik needs exclusive access to ports 80 and 443", port)
		}
	}

	return releaseCurrent, nil
}

func publishedPortOwner(client *ssh.Client, port int) (string, error) {
	lines, err := captureStdoutLines(client, fmt.Sprintf("%s ps --filter publish=%d --format '{{.Names}}'", client.DockerBin(), port))
	if err != nil {
		return "", err
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			return line, nil
		}
	}
	return "", nil
}

func isHostPortBusy(client *ssh.Client, port int) (bool, error) {
	// Try ss first; fall back to netstat if ss is unavailable.
	ssCmd := fmt.Sprintf("ss -ltnH '( sport = :%d )'", port)
	lines, err := captureStdoutLines(client, ssCmd)
	if err != nil {
		// ss unavailable — verify netstat exists, then use it.
		// "command -v netstat" fails if netstat is not installed.
		if _, checkErr := captureStdoutLines(client, "command -v netstat"); checkErr != nil {
			return false, fmt.Errorf("cannot check port %d: neither ss nor netstat available", port)
		}
		// netstat exists; grep may exit 1 on no match, so wrap with || true.
		netstatCmd := fmt.Sprintf("netstat -ltn 2>/dev/null | { grep ':%d ' || true; }", port)
		lines, err = captureStdoutLines(client, netstatCmd)
		if err != nil {
			return false, fmt.Errorf("cannot check port %d: netstat failed: %w", port, err)
		}
	}
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			return true, nil
		}
	}
	return false, nil
}

func captureStdoutLines(client *ssh.Client, command string) ([]string, error) {
	stdoutCh, stderrCh, done := client.StreamCommand(command)

	var stdoutLines []string
	var stderrLines []string
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for line := range stdoutCh {
			stdoutLines = append(stdoutLines, line)
		}
	}()

	go func() {
		defer wg.Done()
		for line := range stderrCh {
			stderrLines = append(stderrLines, line)
		}
	}()

	wg.Wait()

	if err := <-done; err != nil {
		if len(stderrLines) > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.Join(stderrLines, "; "))
		}
		return nil, err
	}

	return stdoutLines, nil
}
