package jobs

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/WahyuS002/uploy/broker"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/ssh"
)

type DeployConfig struct {
	DeploymentID  string
	Image         string
	ContainerName string
	Port          int
	EnvVars       []db.EnvPair
	Server        ssh.ServerConfig
}

func appendLog(ctx context.Context, deploymentID, msg, logType string) {
	if err := db.AppendLog(ctx, deploymentID, msg, logType); err != nil {
		log.Printf("AppendLog deploymentID=%s error: %v", deploymentID, err)
	}
}

func failDeploy(deploymentID, msg string) {
	log.Println(msg)

	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	appendLog(cleanupCtx, deploymentID, msg, "stderr")

	if err := db.SetDeploymentStatus(cleanupCtx, deploymentID, "failed"); err != nil {
		log.Printf("SetDeploymentStatus deploymentID=%s error: %v", deploymentID, err)
		return
	}

	appendLog(cleanupCtx, deploymentID, "deployment failed", "stderr")
	broker.PublishDone(deploymentID, "failed")
}

func finishDeploy(deploymentID, status string) {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.SetDeploymentStatus(cleanupCtx, deploymentID, status); err != nil {
		log.Printf("SetDeploymentStatus deploymentID=%s error: %v", deploymentID, err)
		return
	}

	appendLog(cleanupCtx, deploymentID, fmt.Sprintf("deployment %s", status), "stdout")
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

	appendLog(ctx, cfg.DeploymentID, "connecting to server...", "stdout")

	client, err := ssh.NewClient(cfg.Server)
	if err != nil {
		failDeploy(cfg.DeploymentID, "SSH connection failed: "+err.Error())
		return
	}
	defer client.Close()

	// step 1: docker pull
	if !runStep(ctx, client, cfg.DeploymentID, "docker pull "+cfg.Image) {
		return
	}

	// step 2: stop old container (ignore error — mungkin belum ada)
	stopCmd := fmt.Sprintf("docker stop %s 2>/dev/null || true", cfg.ContainerName)
	if !runStep(ctx, client, cfg.DeploymentID, stopCmd) {
		return
	}

	// step 3: remove old container (ignore error — mungkin belum ada)
	rmCmd := fmt.Sprintf("docker rm %s 2>/dev/null || true", cfg.ContainerName)
	if !runStep(ctx, client, cfg.DeploymentID, rmCmd) {
		return
	}

	// step 4: docker run dengan env vars
	if !runStep(ctx, client, cfg.DeploymentID, buildDockerRunCmd(cfg)) {
		return
	}

	finishDeploy(cfg.DeploymentID, "success")
}

func buildDockerRunCmd(cfg DeployConfig) string {
	args := fmt.Sprintf("docker run -d --name %s -p %d:80", cfg.ContainerName, cfg.Port)

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
			appendLog(ctx, deploymentID, line, "stdout")
		}
	}()
	go func() {
		defer wg.Done()
		for line := range stderrCh {
			appendLog(ctx, deploymentID, line, "stderr")
		}
	}()

	wg.Wait()

	if err := <-done; err != nil {
		failDeploy(deploymentID, fmt.Sprintf("command failed: %v", err))
		return false
	}
	return true
}
