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
		ContainerPort: 80,
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
		ContainerPort: 6379,
		HostPort:      6379,
	})
	if !strings.Contains(direct, "-p 6379:6379") {
		t.Errorf("direct mode did not publish the container port: %s", direct)
	}

	proxied := buildDockerRunCmd("docker", DeployConfig{
		ContainerName: "web-app",
		Image:         "ghcr.io/owner/repo:tag",
		ContainerPort: 3000,
		HostPort:      3000,
		Domains:       []string{"example.com"},
	})
	if strings.Contains(proxied, "traefik.http") {
		t.Errorf("proxy mode should be routed by the file provider, not container labels: %s", proxied)
	}
	if strings.Contains(proxied, "-p ") {
		t.Errorf("proxy mode should not publish a host port: %s", proxied)
	}
}

func TestBuildDockerRunCmdAddsDeploymentOwnershipLabels(t *testing.T) {
	cmd := buildDockerRunCmd("docker", DeployConfig{
		DeploymentID:  "deployment-1",
		ServiceID:     "service-1",
		ContainerName: "web-app-deadbeef",
		Image:         "nginx",
		ContainerPort: 80,
		Domains:       []string{"example.com"},
	})
	for _, label := range []string{"--label uploy.service_id=service-1", "--label uploy.deployment_id=deployment-1"} {
		if !strings.Contains(cmd, label) {
			t.Errorf("missing ownership label %q: %s", label, cmd)
		}
	}
}

func TestDeploymentContainerNameUsesShortDeploymentID(t *testing.T) {
	if got := DeploymentContainerName("web-app", "12345678-aaaa-bbbb"); got != "web-app-12345678" {
		t.Fatalf("DeploymentContainerName = %q", got)
	}
}

// A service with no domains and no host port is reachable by other services and
// by nothing else. Publishing it anyway is how a database ends up on the open
// internet without anyone choosing that.
func TestBuildDockerRunCmdKeepsUnpublishedServiceInternal(t *testing.T) {
	internal := buildDockerRunCmd("docker", DeployConfig{
		ContainerName: "postgres-app",
		Image:         "postgres:16",
		ContainerPort: 5432,
		// no HostPort, no Domains
	})
	if strings.Contains(internal, "-p ") {
		t.Errorf("unpublished service was published to the host: %s", internal)
	}
	if !strings.Contains(internal, "--network uploy") {
		t.Errorf("unpublished service is not on the shared network, so nothing can reach it: %s", internal)
	}
}

// Every container joins the shared network, published or not — that is what
// lets one service reach another by name.
func TestBuildDockerRunCmdAlwaysJoinsNetwork(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  DeployConfig
	}{
		{"published", DeployConfig{ContainerName: "a", Image: "redis", ContainerPort: 6379, HostPort: 6379}},
		{"proxied", DeployConfig{ContainerName: "b", Image: "nginx", ContainerPort: 80, Domains: []string{"example.com"}}},
		{"internal", DeployConfig{ContainerName: "c", Image: "postgres:16", ContainerPort: 5432}},
	} {
		cmd := buildDockerRunCmd("docker", tc.cfg)
		if !strings.Contains(cmd, "--network uploy") {
			t.Errorf("%s is not on the shared network: %s", tc.name, cmd)
		}
	}
}

// A server reboot must not silently take every deployed service down with it,
// so both paths have to set a restart policy.
func TestBuildDockerRunCmdSetsRestartPolicy(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  DeployConfig
	}{
		{"direct", DeployConfig{ContainerName: "redis-app", Image: "redis:7-alpine", ContainerPort: 6379, HostPort: 6379}},
		{"proxied", DeployConfig{ContainerName: "web-app", Image: "nginx", ContainerPort: 3000, HostPort: 3000, Domains: []string{"example.com"}}},
	} {
		cmd := buildDockerRunCmd("docker", tc.cfg)
		if !strings.Contains(cmd, "--restart unless-stopped") {
			t.Errorf("%s mode has no restart policy: %s", tc.name, cmd)
		}
	}
}
