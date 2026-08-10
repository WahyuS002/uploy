package proxy

import (
	"strings"
	"testing"
)

func TestRouteConfigPinsSingleCandidate(t *testing.T) {
	config := routeConfig("service-1", []string{"app.example.com", "www.example.com"}, "app-deadbeef", 3000)
	for _, want := range []string{
		`rule: "Host(` + "`app.example.com`" + `) || Host(` + "`www.example.com`" + `)"`,
		"priority: 1000",
		`url: "http://app-deadbeef:3000"`,
	} {
		if !strings.Contains(config, want) {
			t.Errorf("route config missing %q:\n%s", want, config)
		}
	}
	if strings.Count(config, "url:") != 1 {
		t.Fatalf("route must contain exactly one backend:\n%s", config)
	}
}

func TestRouteBackend(t *testing.T) {
	if got := routeBackend([]byte(`{"loadBalancer":{"servers":[{"url":"http://app-deadbeef:3000"}]}}`)); got != "http://app-deadbeef:3000" {
		t.Fatalf("routeBackend() = %q", got)
	}
}

func TestHostnamesFromRule(t *testing.T) {
	got := hostnamesFromRule("Host(`app.example.com`) || Host(`www.example.com`)")
	want := []string{"app.example.com", "www.example.com"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("hostnamesFromRule() = %v, want %v", got, want)
	}
}
