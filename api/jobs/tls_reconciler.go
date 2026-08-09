package jobs

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/WahyuS002/uploy/crypto"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/ssh"
)

const domainCheckInterval = 60 * time.Second

// When the loop below will next look, as unix milliseconds. A pending domain
// cannot change status between two ticks, so this is the only moment worth
// waiting for — and a client that knows it can say so instead of guessing.
//
// Atomic because the reconciler writes it while request handlers read it.
var nextDomainCheck atomic.Int64

// NextDomainCheck reports when pending domains are next examined. The zero time
// means the reconciler is not running, and callers should say nothing rather
// than invent a schedule.
func NextDomainCheck() time.Time {
	ms := nextDomainCheck.Load()
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}

// StartDomainReconciler runs a background loop that checks pending
// application_domains for TLS certificate readiness per exact hostname.
func StartDomainReconciler(ctx context.Context) {
	ticker := time.NewTicker(domainCheckInterval)
	defer ticker.Stop()
	nextDomainCheck.Store(time.Now().Add(domainCheckInterval).UnixMilli())

	for {
		select {
		case <-ctx.Done():
			// Nothing is coming, so stop promising that something is.
			nextDomainCheck.Store(0)
			return
		case <-ticker.C:
			reconcilePendingDomains(ctx)
			// Published after the pass, not before it: a run can take up to 30s
			// of SSH, and counting down to a moment that has already gone by is
			// worse than counting down to one slightly late.
			nextDomainCheck.Store(time.Now().Add(domainCheckInterval).UnixMilli())
		}
	}
}

func reconcilePendingDomains(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()

	domains, err := db.ListDomainsToReconcile(ctx)
	if err != nil {
		log.Printf("Domain reconciler: list domains: %v", err)
		return
	}

	// DNS first, and for most passes it is the only thing that happens.
	//
	// A certificate in acme.json survives the record that earned it, so it can
	// only ever say the name worked once — which is how a hostname with no DNS
	// record at all came to read as HTTPS active. Whether it resolves here is the
	// part that changes when someone edits their DNS, so it decides first, and a
	// domain that fails it never reaches the server at all.
	var needsCertificate []db.PendingDomain
	for _, d := range domains {
		if domainPointsAtServer(ctx, d.Domain, d.Host) {
			// Only the ones with something left to prove are worth an SSH session.
			if d.Status == "ready" {
				if err := db.TouchDomainChecked(ctx, d.ID); err != nil {
					log.Printf("Domain reconciler: touch domain %s: %v", d.ID, err)
				}
				continue
			}
			needsCertificate = append(needsCertificate, d)
			continue
		}

		// Ready and no longer resolving here: the certificate is still on disk and
		// still proves nothing, so the domain goes back to waiting rather than
		// going on claiming HTTPS for a name that answers nowhere.
		if d.Status == "ready" {
			log.Printf("Domain reconciler: domain %s (%s) no longer resolves to %s", d.ID, d.Domain, d.Host)
			if err := db.SetDomainPending(ctx, d.ID); err != nil {
				log.Printf("Domain reconciler: demote domain %s: %v", d.ID, err)
			}
			continue
		}

		// Still waiting for the record to appear, which is the ordinary state of a
		// domain someone added a minute ago. Nothing to say about it beyond that it
		// was looked at.
		if err := db.TouchDomainChecked(ctx, d.ID); err != nil {
			log.Printf("Domain reconciler: touch domain %s: %v", d.ID, err)
		}
	}
	domains = needsCertificate
	if len(domains) == 0 {
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
			certPresent, err := promoteDomainIfCertificateReady(ctx, client, sk.Host, d.ID, d.Domain)
			if certPresent {
				if err != nil {
					log.Printf("Domain reconciler: promote domain %s (%s) failed: %v", d.ID, d.Domain, err)
				} else {
					log.Printf("Domain reconciler: domain %s (%s) is ready", d.ID, d.Domain)
				}
			}
		}

		client.Close()
	}
}
