package jobs

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"runtime/debug"
	"time"

	"github.com/WahyuS002/uploy/db"
)

func RunNginx(deploymentID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered deploymentID=%v: %v\n%s", deploymentID, r, debug.Stack())

			dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer dbCancel()

			if dbErr := db.SetDeploymentStatus(dbCtx, deploymentID, "failed"); dbErr != nil {
				log.Printf("error SetDeploymentStatus in recover deploymentID=%v", deploymentID)
			}

			if dbErr := db.AppendLog(deploymentID, fmt.Sprintf("panic: %v", r)); dbErr != nil {
				log.Printf("error AppendLog in recover deploymentID=%v", deploymentID)
			}
		}
	}()
	pullCtx, pullCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer pullCancel()

	err := exec.CommandContext(pullCtx, "docker", "pull", "nginx:latest").Run()

	status := "success"
	if err != nil {
		status = "failed"
		log.Println("Docker pull nginx:latest err: ", err)
		db.AppendLog(deploymentID, fmt.Sprintf("docker pull failed: %v", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbErr := db.SetDeploymentStatus(ctx, deploymentID, status)
	if dbErr != nil {
		log.Println("error SetDeploymentStatus: ", dbErr)
	}

	db.AppendLog(deploymentID, fmt.Sprintf("deployment %s", status))
}
