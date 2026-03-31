package jobs

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/WahyuS002/uploy/broker"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/proxy"
	"github.com/WahyuS002/uploy/ssh"
)

const proxyContainerName = "uploy-proxy"

type DeployConfig struct {
	DeploymentID  string
	ApplicationID string
	Image         string
	ContainerName string
	Port          int
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

	currentContainerRemoved := false

	// step 2: if app has domains, ensure proxy is running
	if hasDomains {
		appendLog(ctx, cfg.DeploymentID, "checking proxy ports...", "stdout", "proxy_setup")
		releaseCurrent, err := checkProxyPortConflicts(client, cfg.ContainerName)
		if err != nil {
			errMsg := err.Error()
			if updateErr := db.SetServerProxyError(ctx, cfg.ServerID, "port_conflict", errMsg); updateErr != nil {
				log.Printf("SetServerProxyError error: %v", updateErr)
			}
			failDeploy(cfg.DeploymentID, "Proxy setup failed: "+errMsg)
			return
		}
		if releaseCurrent {
			appendLog(ctx, cfg.DeploymentID, "releasing current container from reserved proxy ports...", "stdout", "proxy_setup")
			if !stopAndRemoveContainer(ctx, client, cfg.DeploymentID, cfg.ContainerName) {
				return
			}
			currentContainerRemoved = true
		}

		appendLog(ctx, cfg.DeploymentID, "ensuring reverse proxy (Traefik)...", "stdout", "proxy_setup")
		proxyProgress := func(msg string) {
			appendLog(ctx, cfg.DeploymentID, msg, "stdout", "proxy_setup")
		}
		if err := proxy.EnsureProxy(client, proxyProgress); err != nil {
			errMsg := err.Error()
			if updateErr := db.SetServerProxyError(ctx, cfg.ServerID, "degraded", errMsg); updateErr != nil {
				log.Printf("SetServerProxyError error: %v", updateErr)
			}
			failDeploy(cfg.DeploymentID, "Proxy setup failed: "+errMsg)
			return
		}
	}

	// step 3: stop/remove old container unless already removed during proxy migration
	if !currentContainerRemoved {
		appendLog(ctx, cfg.DeploymentID, "stopping existing container...", "stdout", "stop_container")
		if !stopAndRemoveContainer(ctx, client, cfg.DeploymentID, cfg.ContainerName) {
			return
		}
	}

	// step 4: docker run with env vars
	appendLog(ctx, cfg.DeploymentID, "starting application container...", "stdout", "start_container")
	if !runStep(ctx, client, cfg.DeploymentID, buildDockerRunCmd(docker, cfg)) {
		return
	}

	// step 5: after app container is running, reconcile proxy and domain status.
	if hasDomains {
		// Mark server proxy infra as ready
		if err := db.SetServerProxyReady(ctx, cfg.ServerID, "ready"); err != nil {
			log.Printf("SetServerProxyReady error: %v", err)
		}

		// Only poll domains that are not yet ready (pending or error).
		// Domains already marked ready keep their status — certs persist across redeploys.
		domainList, err := db.ListDomainsByApplication(ctx, cfg.ApplicationID)
		if err != nil {
			log.Printf("ListDomainsByApplication error: %v", err)
		}

		unresolvedDomains := make(map[string]string) // domain -> domainID
		for _, d := range domainList {
			if d.Status != "ready" {
				unresolvedDomains[d.Domain] = d.ID
			}
		}

		if len(unresolvedDomains) > 0 {
			appendLog(ctx, cfg.DeploymentID, "checking HTTPS certificate status...", "stdout", "tls_cert")

			for i := 0; i < 6 && len(unresolvedDomains) > 0; i++ {
				if i > 0 {
					time.Sleep(10 * time.Second)
				}
				for domain, domainID := range unresolvedDomains {
					if proxy.HasCertificateForHostname(client, domain) {
						if err := db.SetDomainReady(ctx, domainID); err != nil {
							log.Printf("SetDomainReady %s error: %v", domain, err)
						} else {
							appendLog(ctx, cfg.DeploymentID, fmt.Sprintf("HTTPS is ready for %s", domain), "stdout", "tls_cert")
						}
						delete(unresolvedDomains, domain)
					}
				}
			}

			if len(unresolvedDomains) > 0 {
				names := make([]string, 0, len(unresolvedDomains))
				for domain := range unresolvedDomains {
					names = append(names, domain)
				}
				sort.Strings(names)
				appendLog(ctx, cfg.DeploymentID, fmt.Sprintf("HTTPS is not ready yet for: %s. The app is deployed and Uploy will keep checking in the background.", strings.Join(names, ", ")), "stdout", "tls_cert")
			}
		}
	}

	finishDeploy(cfg.DeploymentID, "success")
}

func buildDockerRunCmd(docker string, cfg DeployConfig) string {
	var args string

	if len(cfg.Domains) > 0 {
		// Proxy mode: container on "uploy" network, no host port mapping.
		// Traefik forwards to the container's internal port (80).
		args = fmt.Sprintf("%s run -d --name %s --network uploy", docker, cfg.ContainerName)

		routerName := strings.ReplaceAll(cfg.ContainerName, ".", "-")

		// Build Host rule for all domains
		hostRules := make([]string, len(cfg.Domains))
		for i, domain := range cfg.Domains {
			hostRules[i] = fmt.Sprintf("Host(`%s`)", domain)
		}
		hostRule := strings.Join(hostRules, " || ")

		args += " --label traefik.enable=true"
		args += fmt.Sprintf(" --label 'traefik.http.routers.%s.rule=%s'", routerName, hostRule)
		args += fmt.Sprintf(" --label traefik.http.routers.%s.entrypoints=https", routerName)
		args += fmt.Sprintf(" --label traefik.http.routers.%s.tls=true", routerName)
		args += fmt.Sprintf(" --label traefik.http.routers.%s.tls.certresolver=letsencrypt", routerName)
		args += fmt.Sprintf(" --label traefik.http.services.%s.loadbalancer.server.port=80", routerName)
	} else {
		// Direct mode: map host port to container port 80
		args = fmt.Sprintf("%s run -d --name %s -p %d:80", docker, cfg.ContainerName, cfg.Port)
	}

	for _, env := range cfg.EnvVars {
		escaped := strings.ReplaceAll(env.Value, "'", "'\\''")
		args += fmt.Sprintf(" --env '%s=%s'", env.Key, escaped)
	}

	args += " " + cfg.Image
	return args
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
