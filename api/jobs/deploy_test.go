package jobs

import (
	"strings"
	"testing"
)

// The port a user picks is the port the image listens on, so it has to reach
// the container on both paths — published as-is in direct mode, and handed to
// Traefik in proxy mode. Hardcoding 80 here silently broke redis and postgres.
func TestBuildDockerRunCmdUsesContainerPort(t *testing.T) {
	direct := buildDockerRunCmd("docker", DeployConfig{
		ContainerName: "redis-app",
		Image:         "redis:7-alpine",
		Port:          6379,
	})
	if !strings.Contains(direct, "-p 6379:6379") {
		t.Errorf("direct mode did not publish the container port: %s", direct)
	}

	proxied := buildDockerRunCmd("docker", DeployConfig{
		ContainerName: "web-app",
		Image:         "ghcr.io/owner/repo:tag",
		Port:          3000,
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
		{"direct", DeployConfig{ContainerName: "redis-app", Image: "redis:7-alpine", Port: 6379}},
		{"proxied", DeployConfig{ContainerName: "web-app", Image: "nginx", Port: 3000, Domains: []string{"example.com"}}},
	} {
		cmd := buildDockerRunCmd("docker", tc.cfg)
		if !strings.Contains(cmd, "--restart unless-stopped") {
			t.Errorf("%s mode has no restart policy: %s", tc.name, cmd)
		}
	}
}
