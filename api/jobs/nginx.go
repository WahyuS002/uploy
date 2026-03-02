package jobs

import (
	"context"
	"log"
	"os/exec"
	"time"

	"github.com/WahyuS002/uploy/db"
)

func RunNginx(deploymentID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := exec.Command("docker", "pull", "nginx:latest").Run()
	if err != nil {
		dbErr := db.SetDeploymentStatus(ctx, deploymentID, "failed")
		log.Println("update failed status error:", dbErr)
		return
	}

	dbErr := db.SetDeploymentStatus(ctx, deploymentID, "success")
	log.Println("update finished status error:", dbErr)
}
