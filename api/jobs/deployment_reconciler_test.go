package jobs

import (
	"context"
	"errors"
	"testing"
)

type probeResult struct {
	output string
	err    error
}

type probeRunner struct {
	results  []probeResult
	commands []string
}

func (r *probeRunner) DockerBin() string {
	return "docker"
}

func (r *probeRunner) Run(_ context.Context, command string) (string, error) {
	r.commands = append(r.commands, command)
	result := r.results[0]
	r.results = r.results[1:]
	return result.output, result.err
}

func TestIsHealthyOrRunning(t *testing.T) {
	t.Run("uses running state only without a health check", func(t *testing.T) {
		runner := &probeRunner{results: []probeResult{{output: ""}, {output: "true"}}}

		healthy, err := isHealthyOrRunning(context.Background(), runner, "app")
		if err != nil {
			t.Fatalf("isHealthyOrRunning() error = %v", err)
		}
		if !healthy {
			t.Fatal("isHealthyOrRunning() = false; want true")
		}
		if len(runner.commands) != 2 {
			t.Fatalf("command count = %d; want 2", len(runner.commands))
		}
	})

	t.Run("unhealthy container does not fall back to running", func(t *testing.T) {
		runner := &probeRunner{results: []probeResult{{output: "unhealthy"}}}

		healthy, err := isHealthyOrRunning(context.Background(), runner, "app")
		if err != nil {
			t.Fatalf("isHealthyOrRunning() error = %v", err)
		}
		if healthy {
			t.Fatal("isHealthyOrRunning() = true; want false")
		}
		if len(runner.commands) != 1 {
			t.Fatalf("command count = %d; want 1", len(runner.commands))
		}
	})

	t.Run("health inspect error is propagated", func(t *testing.T) {
		runErr := errors.New("ssh disconnected")
		runner := &probeRunner{results: []probeResult{{err: runErr}}}

		healthy, err := isHealthyOrRunning(context.Background(), runner, "app")
		if healthy {
			t.Fatal("isHealthyOrRunning() = true; want false")
		}
		if !errors.Is(err, runErr) {
			t.Fatalf("isHealthyOrRunning() error = %v; want wrapped %v", err, runErr)
		}
		if len(runner.commands) != 1 {
			t.Fatalf("command count = %d; want 1", len(runner.commands))
		}
	})
}
