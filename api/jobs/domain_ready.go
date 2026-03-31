package jobs

import (
	"context"

	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/proxy"
	"github.com/WahyuS002/uploy/ssh"
)

// promoteDomainIfCertificateReady checks whether acme.json on the remote
// server already contains a certificate for domain.  If it does, it marks
// the application_domain row as ready.
//
// Return values:
//   - certificatePresent=false, err=nil  → hostname not yet in acme.json
//   - certificatePresent=true,  err=nil  → cert found and DB updated
//   - certificatePresent=true,  err!=nil → cert found but DB update failed
func promoteDomainIfCertificateReady(ctx context.Context, client *ssh.Client, domainID, domain string) (certificatePresent bool, err error) {
	if !proxy.HasCertificateForHostname(client, domain) {
		return false, nil
	}
	if err := db.SetDomainReady(ctx, domainID); err != nil {
		return true, err
	}
	return true, nil
}
