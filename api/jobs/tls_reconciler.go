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

// StartDomainReconciler runs a background loop that checks pending
// application_domains for TLS certificate readiness per exact hostname.
func StartDomainReconciler(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reconcilePendingDomains(ctx)
		}
	}
}

func reconcilePendingDomains(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()

	domains, err := db.ListUnresolvedDomains(ctx)
	if err != nil {
		log.Printf("Domain reconciler: list unresolved domains: %v", err)
		return
	}

	// Group domains by server to reuse SSH connections.
	type serverKey struct {
		Host       string
		Port       int32
		SSHUser    string
		PrivateKey string
	}
	byServer := make(map[string][]db.PendingDomain) // keyed by serverID
	serverConns := make(map[string]serverKey)

	for _, d := range domains {
		byServer[d.ServerID] = append(byServer[d.ServerID], d)
		if _, ok := serverConns[d.ServerID]; !ok {
			pk, err := crypto.Decrypt(d.EncryptedKey)
			if err != nil {
				log.Printf("Domain reconciler: decrypt key for server %s: %v", d.ServerID, err)
				continue
			}
			serverConns[d.ServerID] = serverKey{
				Host:       d.Host,
				Port:       d.ServerPort,
				SSHUser:    d.SSHUser,
				PrivateKey: pk,
			}
		}
	}

	for serverID, serverDomains := range byServer {
		if ctx.Err() != nil {
			return
		}

		sk, ok := serverConns[serverID]
		if !ok {
			continue // key decryption failed earlier
		}

		client, err := ssh.NewClient(ssh.ServerConfig{
			Host:       sk.Host,
			Port:       int(sk.Port),
			User:       sk.SSHUser,
			PrivateKey: sk.PrivateKey,
		})
		if err != nil {
			log.Printf("Domain reconciler: SSH to server %s failed: %v", serverID, err)
			continue
		}

		for _, d := range serverDomains {
			if proxy.HasCertificateForHostname(client, d.Domain) {
				if err := db.SetDomainReady(ctx, d.ID); err != nil {
					log.Printf("Domain reconciler: promote domain %s (%s) failed: %v", d.ID, d.Domain, err)
				} else {
					log.Printf("Domain reconciler: domain %s (%s) is ready", d.ID, d.Domain)
				}
			}
		}

		client.Close()
	}
}
