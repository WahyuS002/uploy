package jobs

import (
	"strings"
	"testing"
)

// The port the image listens on and the port it is published as are two
// different numbers, and the mapping has to keep them in that order.
//
// Both mistakes have shipped: hardcoding :80 broke redis and postgres, and
// then publishing port:port broke every image that listens on 80, because
// nothing inside the container answered on the number it was published as.
func TestBuildDockerRunCmdMapsHostPortToContainerPort(t *testing.T) {
	// nginx listens on 80 and cannot be published there — 80 belongs to Traefik.
	web := buildDockerRunCmd("docker", DeployConfig{
		ContainerName: "nginx-app",
		Image:         "nginx:latest",
		Port:          80,
		HostPort:      9090,
	})
	if !strings.Contains(web, "-p 9090:80") {
		t.Errorf("host port was not mapped to the container port: %s", web)
	}

	// A database is reached on the number its image is known by, so both sides
	// match — but they still travel through HostPort, not by reusing Port.
	direct := buildDockerRunCmd("docker", DeployConfig{
		ContainerName: "redis-app",
		Image:         "redis:7-alpine",
		Port:          6379,
		HostPort:      6379,
	})
	if !strings.Contains(direct, "-p 6379:6379") {
		t.Errorf("direct mode did not publish the container port: %s", direct)
	}

	proxied := buildDockerRunCmd("docker", DeployConfig{
		ContainerName: "web-app",
		Image:         "ghcr.io/owner/repo:tag",
		Port:          3000,
		HostPort:      3000,
		Domains:       []string{"example.com"},
	})
	if !strings.Contains(proxied, "loadbalancer.server.port=3000") {
		t.Errorf("proxy mode did not forward to the container port: %s", proxied)
	}
	if strings.Contains(proxied, "-p ") {
		t.Errorf("proxy mode should not publish a host port: %s", proxied)
	}
}

// A server reboot must not silently take every deployed service down with it,
// so both paths have to set a restart policy.
func TestBuildDockerRunCmdSetsRestartPolicy(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  DeployConfig
	}{
		{"direct", DeployConfig{ContainerName: "redis-app", Image: "redis:7-alpine", Port: 6379, HostPort: 6379}},
		{"proxied", DeployConfig{ContainerName: "web-app", Image: "nginx", Port: 3000, HostPort: 3000, Domains: []string{"example.com"}}},
	} {
		cmd := buildDockerRunCmd("docker", tc.cfg)
		if !strings.Contains(cmd, "--restart unless-stopped") {
			t.Errorf("%s mode has no restart policy: %s", tc.name, cmd)
		}
	}
}
