package jobs

import (
	"context"

	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/proxy"
	"github.com/WahyuS002/uploy/ssh"
)

// promoteDomainIfCertificateReady marks a domain ready when it both resolves to
// this server and has a certificate in acme.json.
//
// Both conditions, because neither is sufficient on its own. acme.json is never
// pruned — a certificate outlives the DNS record that earned it, the router that
// used it, and the domain row itself — so on its own it says only that the name
// worked here once. That is how a hostname whose DNS record had been deleted
// came to read as HTTPS active a minute after being added back.
//
// The check lives here rather than in each caller because both of them promote:
// the deploy job waits for a certificate at the end of a deploy, and the
// reconciler picks up whatever the deploy did not see. A gate in one of them is
// a gate in neither.
//
// Return values:
//   - certificatePresent=false, err=nil  → not resolving here, or not in acme.json
//   - certificatePresent=true,  err=nil  → both hold and the DB was updated
//   - certificatePresent=true,  err!=nil → both hold but the DB update failed
func promoteDomainIfCertificateReady(ctx context.Context, client *ssh.Client, serverHost, domainID, domain string) (certificatePresent bool, err error) {
	if !domainPointsAtServer(ctx, domain, serverHost) {
		return false, nil
	}
	if !proxy.HasCertificateForHostname(client, domain) {
		return false, nil
	}
	if err := db.SetDomainReady(ctx, domainID); err != nil {
		return true, err
	}
	return true, nil
}
