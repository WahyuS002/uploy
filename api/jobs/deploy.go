package jobs

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
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

	// step 2: docker run
	cmd := fmt.Sprintf("docker run -d --name %s -p %d:80 %s",
		cfg.ContainerName, cfg.Port, cfg.Image)
	if !runStep(ctx, client, cfg.DeploymentID, cmd) {
		return
	}

	finishDeploy(cfg.DeploymentID, "success")
}

func runStep(ctx context.Context, client *ssh.Client, deploymentID, command string) bool {
	appendLog(ctx, deploymentID, "→ "+command, "stdout")

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
		failDeploy(deploymentID, fmt.Sprintf("command failed: %s: %v", command, err))
		return false
	}
	return true
}
