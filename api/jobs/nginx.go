package jobs

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"runtime/debug"
	"time"

	"github.com/WahyuS002/uploy/broker"
	"github.com/WahyuS002/uploy/db"
)

func appendLog(ctx context.Context, deploymentID, msg string) {
	if err := db.AppendLog(ctx, deploymentID, msg); err != nil {
		log.Printf("AppendLog deploymentID=%s error: %v", deploymentID, err)
	}
}

func failDeploy(deploymentID, msg string) {
	log.Println(msg)

	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	appendLog(cleanupCtx, deploymentID, msg)

	if err := db.SetDeploymentStatus(cleanupCtx, deploymentID, "failed"); err != nil {
		log.Printf("SetDeploymentStatus deploymentID=%s error: %v", deploymentID, err)
		return
	}

	appendLog(cleanupCtx, deploymentID, "deployment failed")
	broker.PublishDone(deploymentID, "failed")
}

func finishDeploy(deploymentID, status string) {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.SetDeploymentStatus(cleanupCtx, deploymentID, status); err != nil {
		log.Printf("SetDeploymentStatus deploymentID=%s error: %v", deploymentID, err)
		return
	}

	appendLog(cleanupCtx, deploymentID, fmt.Sprintf("deployment %s", status))
	broker.PublishDone(deploymentID, status)
}

func RunNginx(deploymentID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered deploymentID=%v: %v\n%s", deploymentID, r, debug.Stack())

			dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer dbCancel()

			if dbErr := db.SetDeploymentStatus(dbCtx, deploymentID, "failed"); dbErr != nil {
				log.Printf("error SetDeploymentStatus in recover deploymentID=%v", deploymentID)
			}

			if dbErr := db.AppendLog(dbCtx, deploymentID, fmt.Sprintf("panic: %v", r)); dbErr != nil {
				log.Printf("error AppendLog in recover deploymentID=%v", deploymentID)
			}

			broker.PublishDone(deploymentID, "failed")
		}
	}()
	ctx, ctxCancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer ctxCancel()

	pullCtx, pullCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer pullCancel()

	appendLog(ctx, deploymentID, "pulling nginx:latest...")

	cmd := exec.CommandContext(pullCtx, "docker", "pull", "nginx:latest")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		failDeploy(deploymentID, fmt.Sprintf("failed to create stdout pipe: %v", err))
		return
	}
	cmd.Stderr = cmd.Stdout // merge stderr into stdout

	if err := cmd.Start(); err != nil {
		failDeploy(deploymentID, fmt.Sprintf("failed to start docker pull: %v", err))
		return
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		appendLog(ctx, deploymentID, scanner.Text())
	}
	if scanner.Err() != nil && scanner.Err() != io.EOF {
		appendLog(ctx, deploymentID, fmt.Sprintf("error reading output: %v", scanner.Err()))
	}

	err = cmd.Wait()

	status := "success"
	if err != nil {
		status = "failed"
		log.Println("Docker pull nginx:latest err: ", err)
		appendLog(ctx, deploymentID, fmt.Sprintf("docker pull failed: %v", err))
	}

	finishDeploy(deploymentID, status)
}
