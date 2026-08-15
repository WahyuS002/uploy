package docker

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/WahyuS002/uploy/ssh"
)

// CommandRunner executes Docker CLI commands on a remote server.
type CommandRunner interface {
	DockerBin() string
	Run(context.Context, string) (string, error)
}

// ContainerExists reports whether name is registered with Docker, including
// stopped containers. Operational failures are returned instead of being
// reported as a missing container.
func ContainerExists(ctx context.Context, client CommandRunner, name string) (bool, error) {
	filter := "name=^/" + regexp.QuoteMeta(name) + "$"
	command := fmt.Sprintf(
		"%s container ls --all --filter %s --format %s",
		client.DockerBin(),
		ssh.ShellQuote(filter),
		ssh.ShellQuote("{{.ID}}"),
	)
	output, err := client.Run(ctx, command)
	if err != nil {
		return false, fmt.Errorf("list container %q: %w", name, err)
	}
	return strings.TrimSpace(output) != "", nil
}

// ContainerRunning reports whether name is currently running.
func ContainerRunning(ctx context.Context, client CommandRunner, name string) (bool, error) {
	output, err := inspect(ctx, client, name, "{{.State.Running}}")
	if err != nil {
		return false, fmt.Errorf("inspect container %q running state: %w", name, err)
	}
	return strings.TrimSpace(output) == "true", nil
}

// ContainerHealth returns Docker's health status, or an empty string when the
// container has no health check configured.
func ContainerHealth(ctx context.Context, client CommandRunner, name string) (string, error) {
	output, err := inspect(ctx, client, name, "{{if .State.Health}}{{.State.Health.Status}}{{end}}")
	if err != nil {
		return "", fmt.Errorf("inspect container %q health: %w", name, err)
	}
	return strings.TrimSpace(output), nil
}

func inspect(ctx context.Context, client CommandRunner, name, format string) (string, error) {
	command := fmt.Sprintf(
		"%s container inspect --format %s %s",
		client.DockerBin(),
		ssh.ShellQuote(format),
		ssh.ShellQuote(name),
	)
	return client.Run(ctx, command)
}
