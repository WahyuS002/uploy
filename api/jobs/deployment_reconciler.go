package jobs

import (
	"context"
	"errors"
	"fmt"
	"github.com/WahyuS002/uploy/telemetry"
	"time"

	"github.com/WahyuS002/uploy/db"
	dockerapi "github.com/WahyuS002/uploy/docker"
	"github.com/WahyuS002/uploy/proxy"
	"github.com/WahyuS002/uploy/ssh"
)

func StartDeploymentReconciler(ctx context.Context) {
	ids, err := db.ListInProgressDeploymentIDs(ctx)
	if err != nil {
		telemetry.Printf("list interrupted deployments: %v", err)
		return
	}
	for _, deploymentID := range ids {
		go reconcileInterruptedDeployment(ctx, deploymentID)
	}
}

func reconcileInterruptedDeployment(ctx context.Context, deploymentID string) {
	for {
		resolved, err := reconcileDeploymentOnce(ctx, deploymentID)
		if err != nil {
			appendLog(context.Background(), deploymentID, "recovery pending: "+err.Error(), "stderr", "recovery_pending")
		}
		if resolved || ctx.Err() != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(30 * time.Second):
		}
	}
}

func reconcileDeploymentOnce(ctx context.Context, deploymentID string) (bool, error) {
	dep, err := db.GetDeployment(ctx, deploymentID)
	if err != nil {
		return true, err
	}
	if dep.Status != "in_progress" {
		return true, nil
	}
	cfg, err := db.GetDeploymentConfig(ctx, deploymentID)
	if err != nil {
		failDeploy(deploymentID, "deployment interrupted: "+err.Error())
		return true, nil
	}
	svc, err := db.GetServiceWithServer(ctx, dep.ServiceID)
	if err != nil {
		return false, err
	}
	client, err := ssh.NewClient(ssh.ServerConfig{Host: svc.Host, Port: int(svc.ServerPort), User: svc.SSHUser, PrivateKey: svc.PrivateKey})
	if err != nil {
		return false, err
	}
	defer client.Close()
	if err := client.DetectDocker(); err != nil {
		return false, err
	}
	if len(cfg.Domains) > 0 {
		if err := proxy.VerifyRollingReady(ctx, client); err != nil {
			return false, err
		}
	}

	phase, err := db.GetLatestDeploymentPhase(ctx, deploymentID)
	if err != nil {
		return false, err
	}
	candidateName := ContainerNameForDeployment(cfg, deploymentID)
	if phase != "cutover" && phase != "active" && phase != "drain" {
		_ = removeContainer(ctx, client, candidateName)
		failDeploy(deploymentID, "deployment interrupted before traffic cutover")
		return true, nil
	}

	routedContainer, err := proxy.RouteContainer(ctx, client, dep.ServiceID)
	if err != nil {
		return false, err
	}
	candidateHealthy, err := isHealthyOrRunning(ctx, client, candidateName)
	if err != nil {
		return false, fmt.Errorf("inspect candidate container %s: %w", candidateName, err)
	}
	oldDeployment, oldCfg, hasOld, oldErr := previousDeployment(ctx, client, dep.ServiceID, cfg.ContainerName)
	if oldErr != nil {
		return false, oldErr
	}
	oldName := ""
	oldHealthy := false
	if hasOld {
		oldName = ContainerNameForDeployment(oldCfg, oldDeployment.ID)
		oldHealthy, err = isHealthyOrRunning(ctx, client, oldName)
		if err != nil {
			return false, fmt.Errorf("inspect previous container %s: %w", oldName, err)
		}
	}

	if routedContainer == candidateName && candidateHealthy {
		if hasOld && oldHealthy {
			if err := monitorHealthy(ctx, client, candidateName, drainPeriod); err != nil {
				if restoreErr := restorePreviousRoute(ctx, client, dep.ServiceID, oldCfg, oldName, true); restoreErr != nil {
					return false, errors.Join(err, restoreErr)
				}
				_ = removeContainer(ctx, client, candidateName)
				failDeploy(deploymentID, "deployment interrupted during drain; traffic restored to previous deployment")
				return true, nil
			}
			if err := removeContainer(ctx, client, oldName); err != nil {
				return false, err
			}
		}
		finishDeploy(deploymentID, "success")
		return true, nil
	}
	if hasOld && oldHealthy {
		if err := restorePreviousRoute(ctx, client, dep.ServiceID, oldCfg, oldName, true); err != nil {
			return false, err
		}
	} else if err := restorePreviousRoute(ctx, client, dep.ServiceID, db.ServiceConfig{}, "", false); err != nil {
		return false, err
	}
	_ = removeContainer(ctx, client, candidateName)
	failDeploy(deploymentID, "deployment interrupted; traffic restored to previous deployment")
	return true, nil
}

func isHealthyOrRunning(ctx context.Context, client dockerapi.CommandRunner, containerName string) (bool, error) {
	status, err := dockerapi.ContainerHealth(ctx, client, containerName)
	if err != nil {
		return false, err
	}
	if status != "" {
		return status == "healthy", nil
	}
	return dockerapi.ContainerRunning(ctx, client, containerName)
}

func removeContainer(ctx context.Context, client *ssh.Client, containerName string) error {
	_, err := client.Run(ctx, fmt.Sprintf("%s rm -f %s 2>/dev/null || true", client.DockerBin(), containerName))
	return err
}
