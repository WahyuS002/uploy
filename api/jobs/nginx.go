package jobs

import (
	"context"
	"log"
	"os/exec"
	"time"

	"github.com/WahyuS002/uploy/db"
)

func RunNginx(deploymentID string) {
	pullCtx, pullCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer pullCancel()

	err := exec.CommandContext(pullCtx, "docker", "pull", "nginx:latest").Run()

	status := "success"
	if err != nil {
		status = "failed"
		log.Println("Docker pull nginx:latest err: ", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbErr := db.SetDeploymentStatus(ctx, deploymentID, status)
	if dbErr != nil {
		log.Println("error SetDeploymentStatus: ", dbErr)
	}
}
