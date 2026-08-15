package docker

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeRunner struct {
	dockerBin string
	output    string
	err       error
	command   string
}

func (r *fakeRunner) DockerBin() string {
	return r.dockerBin
}

func (r *fakeRunner) Run(_ context.Context, command string) (string, error) {
	r.command = command
	return r.output, r.err
}

func TestContainerExists(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		runner := &fakeRunner{dockerBin: "docker", output: "abc123"}

		exists, err := ContainerExists(context.Background(), runner, "uploy-monitor")
		if err != nil {
			t.Fatalf("ContainerExists() error = %v", err)
		}
		if !exists {
			t.Fatal("ContainerExists() = false; want true")
		}
		if got, want := runner.command, "docker container ls --all --filter 'name=^/uploy-monitor$' --format '{{.ID}}'"; got != want {
			t.Fatalf("command = %q; want %q", got, want)
		}
	})

	t.Run("absent", func(t *testing.T) {
		runner := &fakeRunner{dockerBin: "docker"}

		exists, err := ContainerExists(context.Background(), runner, "uploy-monitor")
		if err != nil {
			t.Fatalf("ContainerExists() error = %v", err)
		}
		if exists {
			t.Fatal("ContainerExists() = true; want false")
		}
	})

	t.Run("operational error", func(t *testing.T) {
		runErr := errors.New("permission denied")
		runner := &fakeRunner{dockerBin: "sudo -n docker", err: runErr}

		exists, err := ContainerExists(context.Background(), runner, "uploy-monitor")
		if exists {
			t.Fatal("ContainerExists() = true; want false")
		}
		if !errors.Is(err, runErr) {
			t.Fatalf("ContainerExists() error = %v; want wrapped %v", err, runErr)
		}
	})
}

func TestContainerRunning(t *testing.T) {
	runner := &fakeRunner{dockerBin: "docker", output: "true"}

	running, err := ContainerRunning(context.Background(), runner, "proxy")
	if err != nil {
		t.Fatalf("ContainerRunning() error = %v", err)
	}
	if !running {
		t.Fatal("ContainerRunning() = false; want true")
	}
	if got, want := runner.command, "docker container inspect --format '{{.State.Running}}' 'proxy'"; got != want {
		t.Fatalf("command = %q; want %q", got, want)
	}
}

func TestContainerHealth(t *testing.T) {
	t.Run("health status", func(t *testing.T) {
		runner := &fakeRunner{dockerBin: "docker", output: "healthy\n"}

		status, err := ContainerHealth(context.Background(), runner, "app")
		if err != nil {
			t.Fatalf("ContainerHealth() error = %v", err)
		}
		if status != "healthy" {
			t.Fatalf("ContainerHealth() = %q; want healthy", status)
		}
	})

	t.Run("no health check", func(t *testing.T) {
		runner := &fakeRunner{dockerBin: "docker"}

		status, err := ContainerHealth(context.Background(), runner, "app")
		if err != nil {
			t.Fatalf("ContainerHealth() error = %v", err)
		}
		if status != "" {
			t.Fatalf("ContainerHealth() = %q; want empty status", status)
		}
	})

	t.Run("operational error", func(t *testing.T) {
		runErr := errors.New("ssh disconnected")
		runner := &fakeRunner{dockerBin: "docker", err: runErr}

		_, err := ContainerHealth(context.Background(), runner, "app")
		if !errors.Is(err, runErr) {
			t.Fatalf("ContainerHealth() error = %v; want wrapped %v", err, runErr)
		}
		if !strings.Contains(runner.command, "container inspect") {
			t.Fatalf("command = %q; want container inspect", runner.command)
		}
	})
}
