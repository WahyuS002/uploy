package jobs

import (
	"context"
	"log"
	"time"

	"github.com/WahyuS002/uploy/crypto"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/proxy"
	"github.com/WahyuS002/uploy/ssh"
)

// StartTLSReconciler runs a background loop that promotes servers stuck in
// tls_pending to ready once ACME certificates appear. This covers cases where
// certificate issuance takes longer than the 60s deploy-time poll window.
func StartTLSReconciler(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reconcileTLSPending(ctx)
		}
	}
}

func reconcileTLSPending(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()

	servers, err := db.ListTLSPendingServers(ctx)
	if err != nil {
		log.Printf("TLS reconciler: list pending servers: %v", err)
		return
	}

	for _, s := range servers {
		if ctx.Err() != nil {
			return
		}

		pk, err := crypto.Decrypt(s.EncryptedKey)
		if err != nil {
			log.Printf("TLS reconciler: decrypt key for server %s: %v", s.ID, err)
			continue
		}

		client, err := ssh.NewClient(ssh.ServerConfig{
			Host:       s.Host,
			Port:       int(s.Port),
			User:       s.SSHUser,
			PrivateKey: pk,
		})
		if err != nil {
			log.Printf("TLS reconciler: SSH to %s failed: %v", s.ID, err)
			continue
		}

		if proxy.HasCertificates(client) {
			if err := db.SetServerProxyReady(ctx, s.ID, "ready"); err != nil {
				log.Printf("TLS reconciler: promote %s failed: %v", s.ID, err)
			} else {
				log.Printf("TLS reconciler: promoted server %s to ready", s.ID)
			}
		}

		client.Close()
	}
}
