package proxy

import (
	"fmt"
	"strings"
	"sync"

	"github.com/WahyuS002/uploy/ssh"
)

const networkName = "uploy"
const proxyBaseDir = "/data/uploy/proxy"
const composeFilePath = proxyBaseDir + "/docker-compose.yaml"
const proxyContainerName = "uploy-proxy"
const traefikImage = "traefik:v3.6"

// EnsureProxy ensures Traefik compose file exists and the service is running.
func EnsureProxy(client *ssh.Client) error {
	docker := client.DockerBin()

	// 1. docker compose must be available
	if err := runSimple(client, docker+" compose version >/dev/null 2>&1"); err != nil {
		return fmt.Errorf("docker compose not available: %w", err)
	}

	// 2. Create Docker network
	if err := runIgnoreError(client, fmt.Sprintf(
		"%s network create %s 2>/dev/null || true", docker, networkName,
	)); err != nil {
		return fmt.Errorf("create network: %w", err)
	}

	// 3. Setup directory + acme.json (each command retries with sudo -n on failure)
	setupCmds := []string{
		fmt.Sprintf("mkdir -p %s", proxyBaseDir),
		fmt.Sprintf("touch %s/acme.json", proxyBaseDir),
		fmt.Sprintf("chmod 600 %s/acme.json", proxyBaseDir),
	}
	for _, cmd := range setupCmds {
		if err := runElevated(client, cmd); err != nil {
			return fmt.Errorf("setup dirs: %w", err)
		}
	}

	// 4. Write docker-compose.yaml
	composeContent := fmt.Sprintf(`services:
  traefik:
    image: %s
    container_name: %s
    restart: unless-stopped
    networks:
      - %s
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - %s:/traefik
    command:
      - --providers.docker=true
      - --providers.docker.exposedbydefault=false
      - --providers.docker.network=%s
      - --entrypoints.http.address=:80
      - --entrypoints.https.address=:443
      - --certificatesresolvers.letsencrypt.acme.httpchallenge=true
      - --certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=http
      - --certificatesresolvers.letsencrypt.acme.storage=/traefik/acme.json
      - --entrypoints.http.http.redirections.entrypoint.to=https
      - --entrypoints.http.http.redirections.entrypoint.scheme=https

networks:
  %s:
    external: true
`, traefikImage, proxyContainerName, networkName, proxyBaseDir, networkName, networkName)

	writeCmd := fmt.Sprintf("cat <<'EOF' | tee %s >/dev/null\n%s\nEOF", composeFilePath, composeContent)
	writeCmdSudo := fmt.Sprintf("cat <<'EOF' | sudo -n tee %s >/dev/null\n%s\nEOF", composeFilePath, composeContent)
	if err := runSimple(client, writeCmd); err != nil {
		if client.IsRoot() {
			return fmt.Errorf("write compose file: %w", err)
		}
		if err := runSimple(client, writeCmdSudo); err != nil {
			return fmt.Errorf("write compose file: %w", err)
		}
	}

	// 5. Start / reconcile Traefik via Compose
	upCmd := fmt.Sprintf("%s compose -f %s up -d", docker, composeFilePath)
	if err := runSimple(client, upCmd); err != nil {
		return fmt.Errorf("compose up: %w", err)
	}

	// 6. Verify proxy container is running
	running, err := isContainerRunning(client, proxyContainerName)
	if err != nil {
		return fmt.Errorf("check proxy: %w", err)
	}
	if !running {
		return fmt.Errorf("proxy container %s is not running", proxyContainerName)
	}

	return nil
}

func isContainerRunning(client *ssh.Client, name string) (bool, error) {
	stdoutCh, _, done := client.StreamCommand(
		fmt.Sprintf("%s inspect -f '{{.State.Running}}' %s 2>/dev/null || echo false", client.DockerBin(), name),
	)

	var output string
	for line := range stdoutCh {
		output = line
	}
	if err := <-done; err != nil {
		return false, nil
	}

	return output == "true", nil
}

func drainBoth(stdoutCh, stderrCh <-chan string) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); for range stdoutCh {} }()
	go func() { defer wg.Done(); for range stderrCh {} }()
	wg.Wait()
}

// runElevated tries cmd directly; for non-root users, retries with sudo -n on failure.
func runElevated(client *ssh.Client, cmd string) error {
	if err := runSimple(client, cmd); err == nil {
		return nil
	}
	if client.IsRoot() {
		return runSimple(client, cmd)
	}
	return runSimple(client, "sudo -n "+cmd)
}

func runSimple(client *ssh.Client, cmd string) error {
	stdoutCh, stderrCh, done := client.StreamCommand(cmd)

	var stderrLines []string
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); for range stdoutCh {} }()
	go func() {
		defer wg.Done()
		for line := range stderrCh {
			stderrLines = append(stderrLines, line)
		}
	}()
	wg.Wait()

	if err := <-done; err != nil {
		if len(stderrLines) > 0 {
			return fmt.Errorf("%w: %s", err, strings.Join(stderrLines, "; "))
		}
		return err
	}
	return nil
}

// HasCertificates checks if acme.json has certificate data (file size > 2 bytes).
// An empty acme.json is "{}" (2 bytes) or literally empty; certificates make it larger.
func HasCertificates(client *ssh.Client) bool {
	// stat -c %s gives file size in bytes on Linux
	cmd := fmt.Sprintf("stat -c %%s %s/acme.json 2>/dev/null || echo 0", proxyBaseDir)
	stdoutCh, _, done := client.StreamCommand(cmd)
	var output string
	for line := range stdoutCh {
		output = strings.TrimSpace(line)
	}
	<-done
	// "{}" = 2 bytes, empty = 0 bytes; anything larger means certs exist
	return output != "" && output != "0" && output != "2"
}

func runIgnoreError(client *ssh.Client, cmd string) error {
	stdoutCh, stderrCh, done := client.StreamCommand(cmd)
	drainBoth(stdoutCh, stderrCh)
	<-done
	return nil
}
