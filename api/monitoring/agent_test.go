package monitoring

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestPinnedImage(t *testing.T) {
	for _, image := range []string{"ghcr.io/acme/monitor:v1.2.3", "ghcr.io/acme/monitor@sha256:abc"} {
		if !pinnedImage(image) {
			t.Fatalf("expected pinned image: %s", image)
		}
	}
	for _, image := range []string{"ghcr.io/acme/monitor", "ghcr.io/acme/monitor:latest"} {
		if pinnedImage(image) {
			t.Fatalf("expected unpinned image: %s", image)
		}
	}
}

func TestPrivateURL(t *testing.T) {
	if got := PrivateURL("10.0.0.4", 9184); got != "http://10.0.0.4:9184" {
		t.Fatalf("PrivateURL = %q", got)
	}
}

func TestPrivateIP(t *testing.T) {
	for _, value := range []string{"10.0.0.4", "192.168.1.4", "100.64.0.2", "fd00::2"} {
		if !privateIP(value) {
			t.Fatalf("expected private IP: %s", value)
		}
	}
	for _, value := range []string{"127.0.0.1", "0.0.0.0", "8.8.8.8", "example.com"} {
		if privateIP(value) {
			t.Fatalf("expected rejected IP: %s", value)
		}
	}
}

func TestValidateConfigRequiresBothTokens(t *testing.T) {
	config := Config{Image: "ghcr.io/acme/monitor:v1", PrivateAddress: "10.0.0.4", HostPort: 9184, RetentionDays: 7, ControlToken: "control"}
	if err := ValidateConfig(config); err == nil {
		t.Fatal("expected missing reader token error")
	}
}

func TestBuildRunCommand(t *testing.T) {
	command := buildRunCommand("sudo -n docker", Config{
		Image:          "ghcr.io/acme/monitor:v1",
		PrivateAddress: "10.0.0.4",
		HostPort:       9184,
		RetentionDays:  7,
		ControlToken:   "control's token",
		ReaderToken:    "reader token",
	})

	for _, want := range []string{
		"sudo -n docker run -d --name uploy-monitor",
		"-p 10.0.0.4:9184:9184",
		"UPLOY_MONITOR_CONTROL_TOKEN='control'\\''s token'",
		"UPLOY_MONITOR_READER_TOKEN='reader token'",
		"UPLOY_MONITOR_RETENTION_DAYS=7",
		"'ghcr.io/acme/monitor:v1'",
	} {
		if !strings.Contains(command, want) {
			t.Fatalf("buildRunCommand() = %q; missing %q", command, want)
		}
	}
}

type commandResult struct {
	output string
	err    error
}

type sequenceRunner struct {
	results  []commandResult
	commands []string
}

func (r *sequenceRunner) DockerBin() string {
	return "docker"
}

func (r *sequenceRunner) Run(_ context.Context, command string) (string, error) {
	r.commands = append(r.commands, command)
	if len(r.results) == 0 {
		return "", nil
	}
	result := r.results[0]
	r.results = r.results[1:]
	return result.output, result.err
}

func TestEnableStateCleanup(t *testing.T) {
	t.Run("restores a previously running container", func(t *testing.T) {
		runner := &sequenceRunner{results: []commandResult{{output: "id"}, {}, {}}}
		state := enableState{
			oldMoved:      true,
			oldWasRunning: true,
			newStarted:    true,
			rollbackName:  ContainerName + "-rollback",
		}

		if err := state.cleanup(context.Background(), runner, false); err != nil {
			t.Fatalf("cleanup() error = %v", err)
		}
		want := []string{
			"docker container ls --all --filter 'name=^/uploy-monitor$' --format '{{.ID}}'",
			"docker rm -f 'uploy-monitor'",
			"docker rename uploy-monitor-rollback uploy-monitor && docker start uploy-monitor",
		}
		if got := strings.Join(runner.commands, "\n"); got != strings.Join(want, "\n") {
			t.Fatalf("cleanup commands = %q; want %q", runner.commands, want)
		}
	})

	t.Run("removes the rollback container after success", func(t *testing.T) {
		runner := &sequenceRunner{results: []commandResult{{output: "id"}, {}}}
		state := enableState{oldMoved: true, rollbackName: ContainerName + "-rollback"}

		if err := state.cleanup(context.Background(), runner, true); err != nil {
			t.Fatalf("cleanup() error = %v", err)
		}
		want := []string{
			"docker container ls --all --filter 'name=^/uploy-monitor-rollback$' --format '{{.ID}}'",
			"docker rm -f 'uploy-monitor-rollback'",
		}
		if got := strings.Join(runner.commands, "\n"); got != strings.Join(want, "\n") {
			t.Fatalf("cleanup commands = %q; want %q", runner.commands, want)
		}
	})

	t.Run("removes a newly started container when no old one existed", func(t *testing.T) {
		runner := &sequenceRunner{results: []commandResult{{output: "id"}, {}}}
		state := enableState{newStarted: true}

		if err := state.cleanup(context.Background(), runner, false); err != nil {
			t.Fatalf("cleanup() error = %v", err)
		}
		want := []string{
			"docker container ls --all --filter 'name=^/uploy-monitor$' --format '{{.ID}}'",
			"docker rm -f 'uploy-monitor'",
		}
		if got := strings.Join(runner.commands, "\n"); got != strings.Join(want, "\n") {
			t.Fatalf("cleanup commands = %q; want %q", runner.commands, want)
		}
	})

	t.Run("restarts an old container stopped before rename", func(t *testing.T) {
		runner := &sequenceRunner{}
		state := enableState{oldStopped: true}

		if err := state.cleanup(context.Background(), runner, false); err != nil {
			t.Fatalf("cleanup() error = %v", err)
		}
		if got, want := runner.commands, []string{"docker start uploy-monitor"}; strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Fatalf("cleanup commands = %q; want %q", got, want)
		}
	})

	t.Run("returns cleanup failures", func(t *testing.T) {
		runErr := errors.New("ssh disconnected")
		runner := &sequenceRunner{results: []commandResult{{err: runErr}, {}}}
		state := enableState{oldMoved: true, rollbackName: ContainerName + "-rollback"}

		if err := state.cleanup(context.Background(), runner, false); !errors.Is(err, runErr) {
			t.Fatalf("cleanup() error = %v; want wrapped %v", err, runErr)
		}
	})
}
