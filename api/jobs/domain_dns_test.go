package jobs

import "testing"

// The rule on its own, with no resolver in the way: given what the name answered
// and where the server is, does the domain lead here.
func TestAnyAddressMatches(t *testing.T) {
	server := []string{"203.0.113.10"}

	cases := []struct {
		name     string
		resolved []string
		want     bool
	}{
		{"points here", []string{"203.0.113.10"}, true},
		{"points elsewhere", []string{"198.51.100.7"}, false},
		{"one of several points here", []string{"198.51.100.7", "203.0.113.10"}, true},
		// The state that started this: the record was deleted, so the name answers
		// nothing at all — while its certificate sits in acme.json looking valid.
		{"no answer", nil, false},
		{"behind Cloudflare", []string{"104.16.1.1"}, true},
		{"just outside Cloudflare", []string{"104.24.0.1"}, false},
		{"unparseable answer", []string{"not-an-ip"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := anyAddressMatches(tc.resolved, server); got != tc.want {
				t.Errorf("anyAddressMatches(%v, %v) = %v, want %v", tc.resolved, server, got, tc.want)
			}
		})
	}
}

// A server recorded by hostname resolves to several addresses, and the domain
// only has to agree with one of them.
func TestAnyAddressMatchesMultipleServerAddresses(t *testing.T) {
	server := []string{"203.0.113.10", "203.0.113.11"}
	if !anyAddressMatches([]string{"203.0.113.11"}, server) {
		t.Error("a domain pointing at the second server address should match")
	}
	if anyAddressMatches([]string{"203.0.113.12"}, server) {
		t.Error("a neighbouring address is not this server")
	}
}

// The ranges are typed out by hand, so it is worth knowing they parsed at all —
// an empty list would silently turn the Cloudflare allowance into a rejection.
func TestCloudflareRangesParsed(t *testing.T) {
	if len(cloudflareRanges) != 14 {
		t.Fatalf("parsed %d Cloudflare ranges, want 14", len(cloudflareRanges))
	}
}
