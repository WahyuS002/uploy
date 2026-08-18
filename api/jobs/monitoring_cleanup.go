package jobs

import (
	"context"
	"github.com/WahyuS002/uploy/telemetry"
	"time"

	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/monitoring"
	"github.com/WahyuS002/uploy/ssh"
)

const monitoringCleanupInterval = time.Hour

func StartMonitoringCleanup(ctx context.Context) {
	ticker := time.NewTicker(monitoringCleanupInterval)
	defer ticker.Stop()
	for {
		cleanupMonitoringData(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func cleanupMonitoringData(ctx context.Context) {
	servers, err := db.ListMonitoringCleanupDue(ctx)
	if err != nil {
		telemetry.Printf("Monitoring cleanup: list servers: %v", err)
		return
	}
	for _, server := range servers {
		client, err := ssh.NewClient(ssh.ServerConfig{Host: server.Host, Port: int(server.Port), User: server.SSHUser, PrivateKey: server.PrivateKey})
		if err != nil {
			telemetry.Printf("Monitoring cleanup: connect server=%s: %v", server.ID, err)
			continue
		}
		err = monitoring.DeleteLocalData(ctx, client)
		client.Close()
		if err != nil {
			telemetry.Printf("Monitoring cleanup: delete data server=%s: %v", server.ID, err)
			continue
		}
		if err := db.ClearServerMonitoringData(ctx, server.ID); err != nil {
			telemetry.Printf("Monitoring cleanup: clear database server=%s: %v", server.ID, err)
		}
	}
}
