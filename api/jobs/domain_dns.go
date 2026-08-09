package jobs

import (
	"context"
	"net"
	"time"
)

// cloudflareRanges are Cloudflare's published IPv4 proxy ranges.
//
// A domain behind Cloudflare's proxy resolves to Cloudflare, never to the origin,
// so a strict "does this point at my server" check would call every proxied
// domain broken while it works perfectly. Treating these as a pass is the same
// trade Coolify makes, and it is the right way round: telling someone their
// working site is broken costs more than missing a misconfigured one, which the
// certificate check below still catches.
//
// ponytail: hard-coded from cloudflare.com/ips-v4. Fetch the list at runtime if
// Cloudflare ever adds a range and someone notices before we do.
var cloudflareRanges = mustParseCIDRs([]string{
	"173.245.48.0/20", "103.21.244.0/22", "103.22.200.0/22", "103.31.4.0/22",
	"141.101.64.0/18", "108.162.192.0/18", "190.93.240.0/20", "188.114.96.0/20",
	"197.234.240.0/22", "198.41.128.0/17", "162.158.0.0/15", "104.16.0.0/13",
	"172.64.0.0/13", "131.0.72.0/22",
})

func mustParseCIDRs(cidrs []string) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			continue // a typo in a constant should not take the process down
		}
		nets = append(nets, n)
	}
	return nets
}

// resolver asks nameservers rather than the machine Uploy happens to run on.
//
// PreferGo skips cgo's getaddrinfo, and with it the operating system's DNS
// cache. That cache is not a detail: on the machine this was found on, a record
// deleted at the registrar an hour earlier still answered with its old address
// through getaddrinfo, while every nameserver — the domain's own included —
// returned NXDOMAIN. Uploy called the domain live on the strength of it, and it
// was live, from exactly one computer in the world.
//
// The Go resolver reads the same /etc/resolv.conf, so this asks the same
// nameservers the host would; it just declines to be told what they said last
// time.
var resolver = &net.Resolver{PreferGo: true}

// dnsLookupTimeout keeps one unreachable resolver from eating the pass. The
// reconciler gets 30s for every domain on every server, and a name whose
// nameservers are down would otherwise sit on the default resolver timeout.
const dnsLookupTimeout = 3 * time.Second

// domainPointsAtServer reports whether the hostname's DNS leads to this server.
//
// This is the half of "does the domain work" that a certificate cannot answer.
// acme.json keeps a certificate long after the record that earned it is deleted,
// so a name that stopped resolving still looks issued — which is how a domain
// with no DNS record at all came to read as HTTPS active. DNS is the thing that
// actually changes when someone removes a record, so it is the thing to ask.
//
// A lookup failure is reported as "does not point here" rather than as an error
// to retry: NXDOMAIN is the normal answer for a record that has not been created
// yet, which is the state most pending domains are in.
func domainPointsAtServer(ctx context.Context, hostname, serverHost string) bool {
	lookupCtx, cancel := context.WithTimeout(ctx, dnsLookupTimeout)
	defer cancel()

	resolved, err := resolver.LookupHost(lookupCtx, hostname)
	if err != nil {
		return false
	}

	serverIPs := serverAddresses(lookupCtx, serverHost)
	return anyAddressMatches(resolved, serverIPs)
}

// serverAddresses turns however the server is recorded into addresses to compare
// against. Usually it is already an IP; a server added by hostname has to be
// resolved before its domains can be checked against it.
func serverAddresses(ctx context.Context, serverHost string) []string {
	if net.ParseIP(serverHost) != nil {
		return []string{serverHost}
	}
	resolved, err := resolver.LookupHost(ctx, serverHost)
	if err != nil {
		return nil
	}
	return resolved
}

// anyAddressMatches is the comparison on its own, so the rule can be read and
// tested without a resolver in the way.
func anyAddressMatches(resolved, serverIPs []string) bool {
	for _, addr := range resolved {
		ip := net.ParseIP(addr)
		if ip == nil {
			continue
		}
		if inCloudflare(ip) {
			return true
		}
		for _, serverIP := range serverIPs {
			if ip.Equal(net.ParseIP(serverIP)) {
				return true
			}
		}
	}
	return false
}

func inCloudflare(ip net.IP) bool {
	for _, n := range cloudflareRanges {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
