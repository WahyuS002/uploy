package jobs

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/WahyuS002/uploy/db"
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

func TestBuildDockerRunCmdAddsFallbackHealthcheck(t *testing.T) {
	cmd := buildDockerRunCmd("docker", DeployConfig{
		ContainerName:      "nginx-app-candidate",
		Image:              "nginx:latest",
		ContainerPort:      80,
		Domains:            []string{"example.com"},
		HealthcheckCommand: fallbackHealthcheckCommand(80),
	})

	for _, option := range []string{
		"--health-cmd 'curl -fsS http://127.0.0.1:80/ >/dev/null || wget -q -O /dev/null http://127.0.0.1:80/ || exit 1'",
		"--health-interval 5s",
		"--health-timeout 5s",
		"--health-retries 10",
		"--health-start-period 5s",
	} {
		if !strings.Contains(cmd, option) {
			t.Errorf("missing fallback healthcheck option %q: %s", option, cmd)
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

func TestSourceBuildCommandsUsePinnedPlanAndSHA(t *testing.T) {
	sha := strings.Repeat("a", 40)
	source := &SourceDeployment{Owner: "owner", Repo: "demo", SHA: sha, Plan: json.RawMessage(`{"deploy":{"startCommand":"node server.js"}}`)}
	if err := source.validate(); err != nil {
		t.Fatalf("source.validate() error = %v", err)
	}

	workdir := sourceWorkdir("deployment-1")
	fetch := sourceFetchCommand(workdir, source)
	for _, want := range []string{
		"curl -sfL",
		"https://codeload.github.com/owner/demo/tar.gz/" + sha,
		"tar xz",
	} {
		if !strings.Contains(fetch, want) {
			t.Errorf("fetch command missing %q: %s", want, fetch)
		}
	}

	plan := sourcePlanCommand(workdir, source.Plan)
	if !strings.Contains(plan, "railpack-plan.json") || !strings.Contains(plan, string(source.Plan)) {
		t.Errorf("plan command did not transfer plan: %s", plan)
	}

	build := sourceBuildCommand("docker", workdir, "svc-1", "uploy/svc-1:"+sha, source)
	for _, want := range []string{
		"docker buildx build",
		"BUILDKIT_SYNTAX=ghcr.io/railwayapp/railpack-frontend:" + railpackVersion,
		"--build-arg cache-key=svc-1",
		"--output type=docker,name=uploy/svc-1:" + sha,
		workdir + "/demo-" + sha,
	} {
		if !strings.Contains(build, want) {
			t.Errorf("build command missing %q: %s", want, build)
		}
	}
}

func TestSecretsHashUsesStableSortedKeyValueEntries(t *testing.T) {
	envs := []db.EnvPair{
		{Key: "B", Value: "two"},
		{Key: "A", Value: "one"},
	}
	want := sha256.Sum256([]byte("A=oneB=two"))
	if got := SecretsHash(envs); got != fmt.Sprintf("%x", want) {
		t.Fatalf("SecretsHash() = %q, want %x", got, want)
	}
	if got := SecretsHash([]db.EnvPair{{Key: "A", Value: "one"}, {Key: "B", Value: "two"}}); got != SecretsHash(envs) {
		t.Fatal("SecretsHash changed when environment order changed")
	}
	if got := SecretsHash([]db.EnvPair{{Key: "A", Value: "changed"}, {Key: "B", Value: "two"}}); got == SecretsHash(envs) {
		t.Fatal("SecretsHash did not change when an environment value changed")
	}
}

func TestSourceBuildCommandInjectsQuotedSecrets(t *testing.T) {
	sha := strings.Repeat("b", 40)
	source := &SourceDeployment{
		Owner: "owner",
		Repo:  "demo",
		SHA:   sha,
		Plan:  json.RawMessage(`{"deploy":{}}`),
		EnvVars: []db.EnvPair{
			{Key: "Z_LAST", Value: "line one\nline 'two' $three"},
			{Key: "A_FIRST", Value: "plain value"},
		},
	}
	cmd := sourceBuildCommand("docker", "/tmp/work", "svc-1", "uploy/svc-1:"+sha, source)
	for _, want := range []string{
		`A_FIRST='plain value' Z_LAST='line one
line '\''two'\'' $three' docker buildx build`,
		"--build-arg secrets-hash=" + SecretsHash(source.EnvVars),
		"--secret id=A_FIRST,env=A_FIRST",
		"--secret id=Z_LAST,env=Z_LAST",
	} {
		if !strings.Contains(cmd, want) {
			t.Errorf("source build command missing %q: %s", want, cmd)
		}
	}
	if strings.Contains(cmd, "src=") {
		t.Errorf("source build command writes secrets through a file: %s", cmd)
	}
}

func TestSourceBuildCommandWithoutEnvKeepsNormalBuild(t *testing.T) {
	sha := strings.Repeat("c", 40)
	source := &SourceDeployment{Owner: "owner", Repo: "demo", SHA: sha, Plan: json.RawMessage(`{"deploy":{}}`)}
	cmd := sourceBuildCommand("docker", "/tmp/work", "svc-1", "uploy/svc-1:"+sha, source)
	if strings.Contains(cmd, "secrets-hash") || strings.Contains(cmd, "--secret") {
		t.Fatalf("empty environment unexpectedly changed build command: %s", cmd)
	}
	if !strings.Contains(cmd, "docker buildx build") {
		t.Fatalf("normal source build command is missing: %s", cmd)
	}
}

func TestRedactSecretsHidesValuesAndMultilineFragments(t *testing.T) {
	envs := []db.EnvPair{{Key: "TOKEN", Value: "first\nsecond"}, {Key: "URL", Value: "https://example.test"}}
	got := redactSecrets("TOKEN=first\nsecond URL=https://example.test", envs)
	if strings.Contains(got, "first") || strings.Contains(got, "second") || strings.Contains(got, "https://example.test") {
		t.Fatalf("redacted output still contains a secret: %q", got)
	}
}

func TestSourceDeploymentRejectsInvalidEnvironmentName(t *testing.T) {
	sha := strings.Repeat("d", 40)
	source := SourceDeployment{Owner: "owner", Repo: "demo", SHA: sha, Plan: json.RawMessage(`{"deploy":{}}`), EnvVars: []db.EnvPair{{Key: "BAD-NAME", Value: "secret"}}}
	if err := source.validate(); err == nil {
		t.Fatal("source.validate() accepted an invalid environment variable name")
	}
}

func TestDeploymentTimeouts(t *testing.T) {
	if got := deploymentTimeout(DeployConfig{}); got != 10*time.Minute {
		t.Fatalf("image deployment timeout = %v", got)
	}
	if got := deploymentTimeout(DeployConfig{Source: &SourceDeployment{}}); got != 30*time.Minute {
		t.Fatalf("source deployment timeout = %v", got)
	}
}
