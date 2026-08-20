package jobs

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/source"
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
	source := &SourceDeployment{Owner: "owner", Repo: "demo", SHA: sha}
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

	rawPlan := []byte(`{"deploy":{"startCommand":"node server.js"}}`)
	plan := sourcePlanCommand(workdir, rawPlan)
	if !strings.Contains(plan, "railpack-plan.json") || !strings.Contains(plan, string(rawPlan)) {
		t.Errorf("plan command did not transfer plan: %s", plan)
	}

	buildCmd, buildScript := sourceBuildInvocation("docker", workdir, "svc-1", "uploy/svc-1:"+sha, source)
	if buildCmd != "sh -s" {
		t.Errorf("build command = %q, want %q", buildCmd, "sh -s")
	}
	for _, want := range []string{
		"docker buildx build",
		"BUILDKIT_SYNTAX=ghcr.io/railwayapp/railpack-frontend:" + railpackVersion,
		"cache-key=svc-1",
		"type=docker,name=uploy/svc-1:" + sha,
		workdir + "/demo-" + sha,
	} {
		if !strings.Contains(string(buildScript), want) {
			t.Errorf("build script missing %q: %s", want, buildScript)
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

func TestSourceBuildKeepsSecretsOutOfTheCommandLine(t *testing.T) {
	sha := strings.Repeat("b", 40)
	source := &SourceDeployment{
		Owner: "owner",
		Repo:  "demo",
		SHA:   sha,
		EnvVars: []db.EnvPair{
			{Key: "Z_LAST", Value: "line one\nline 'two' $three"},
			{Key: "A_FIRST", Value: "plain value"},
		},
	}
	cmd, script := sourceBuildInvocation("docker", "/tmp/work", "svc-1", "uploy/svc-1:"+sha, source)

	// The command line is the argv of the remote shell: ps shows it and sudo
	// logs it. Nothing secret may appear there.
	for _, secret := range []string{"plain value", "line one", "$three"} {
		if strings.Contains(cmd, secret) {
			t.Errorf("command line leaks %q: %s", secret, cmd)
		}
	}

	for _, want := range []string{
		"A_FIRST='plain value'\nexport A_FIRST\n",
		"Z_LAST='line one\nline '\\''two'\\'' $three'\nexport Z_LAST\n",
		"secrets-hash=" + SecretsHash(source.EnvVars),
		"id=A_FIRST,env=A_FIRST",
		"id=Z_LAST,env=Z_LAST",
	} {
		if !strings.Contains(string(script), want) {
			t.Errorf("build script missing %q: %s", want, script)
		}
	}
	if strings.Contains(string(script), "src=") {
		t.Errorf("source build writes secrets through a file: %s", script)
	}
}

func TestSourceBuildWithoutEnvKeepsNormalBuild(t *testing.T) {
	sha := strings.Repeat("c", 40)
	source := &SourceDeployment{Owner: "owner", Repo: "demo", SHA: sha}
	_, script := sourceBuildInvocation("docker", "/tmp/work", "svc-1", "uploy/svc-1:"+sha, source)
	if strings.Contains(string(script), "secrets-hash") || strings.Contains(string(script), "--secret") {
		t.Fatalf("empty environment unexpectedly changed build script: %s", script)
	}
	if !strings.Contains(string(script), "docker buildx build") {
		t.Fatalf("normal source build is missing: %s", script)
	}
}

// sudo writes the command line it ran into the system log by default, so the
// sudo path is the one where a leak would be permanent rather than transient.
func TestSourceBuildKeepsSecretsOutOfSudoCommandLine(t *testing.T) {
	sha := strings.Repeat("e", 40)
	source := &SourceDeployment{
		Repo:    "demo",
		SHA:     sha,
		EnvVars: []db.EnvPair{{Key: "TOKEN", Value: "secret value"}},
	}
	cmd, script := sourceBuildInvocation("sudo -n docker", "/tmp/work", "svc-1", "uploy/svc-1:"+sha, source)
	if cmd != "sudo -n sh -s" {
		t.Fatalf("sudo build command = %q, want %q", cmd, "sudo -n sh -s")
	}
	if strings.Contains(cmd, "secret value") || strings.Contains(cmd, "TOKEN=") {
		t.Fatalf("sudo command line leaks the secret: %s", cmd)
	}
	if !strings.Contains(string(script), "TOKEN='secret value'\nexport TOKEN\n") {
		t.Fatalf("sudo build script did not export the secret: %s", script)
	}
	if strings.Contains(string(script), "sudo") {
		t.Fatalf("script re-enters sudo instead of running wholly under it: %s", script)
	}
}

func TestRedactSecretsHidesValuesAndMultilineFragments(t *testing.T) {
	envs := []db.EnvPair{{Key: "TOKEN", Value: "first\nsecond"}, {Key: "URL", Value: "https://example.test"}}
	got := redactSecrets("TOKEN=first\nsecond URL=https://example.test QUOTED='first'\\''second'", envs)
	if strings.Contains(got, "first") || strings.Contains(got, "second") || strings.Contains(got, "https://example.test") {
		t.Fatalf("redacted output still contains a secret: %q", got)
	}
}

func TestSourceDeploymentRejectsInvalidEnvironmentName(t *testing.T) {
	sha := strings.Repeat("d", 40)
	source := SourceDeployment{Owner: "owner", Repo: "demo", SHA: sha, EnvVars: []db.EnvPair{{Key: "BAD-NAME", Value: "secret"}}}
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

func TestDescribeSourceNamesTheRuntime(t *testing.T) {
	got := describeSource(source.Info{Provider: "node", RuntimeVersions: map[string]string{"node": "22.11.0"}})
	if got != "node (node 22.11.0)" {
		t.Fatalf("describeSource() = %q", got)
	}
	// Ordering has to be stable or the deploy log reads differently run to run.
	got = describeSource(source.Info{Provider: "node", RuntimeVersions: map[string]string{"node": "22", "bun": "1"}})
	if got != "node (bun 1, node 22)" {
		t.Fatalf("describeSource() = %q", got)
	}
	if got := describeSource(source.Info{Provider: "go"}); got != "go" {
		t.Fatalf("describeSource() without versions = %q", got)
	}
}

// The plan is produced by the deployment now, so a deployment is valid before
// one exists.
func TestSourceDeploymentIsValidWithoutAPlan(t *testing.T) {
	src := SourceDeployment{Owner: "owner", Repo: "demo", SHA: strings.Repeat("a", 40)}
	if err := src.validate(); err != nil {
		t.Fatalf("validate() error = %v", err)
	}
}
