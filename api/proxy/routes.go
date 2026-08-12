package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/WahyuS002/uploy/ssh"
)

var ErrLegacyRouteNotFound = errors.New("legacy route not found")

func SetRoute(client *ssh.Client, serviceID string, domains []string, containerName string, containerPort int) error {
	content := routeConfig(serviceID, domains, containerName, containerPort)
	return setRouteConfig(client, serviceID, content)
}

func SetMonitoringRoute(client *ssh.Client, serviceID string, domains []string, containerName string, containerPort int) error {
	content := monitoringRouteConfig(serviceID, domains, containerName, containerPort)
	return setRouteConfig(client, serviceID, content)
}

func setRouteConfig(client *ssh.Client, serviceID, content string) error {
	path := fmt.Sprintf("%s/dynamic/%s.yaml", proxyBaseDir, serviceID)
	tmp := path + ".tmp"
	write := fmt.Sprintf("cat <<'EOF' | tee %s >/dev/null\n%sEOF\nmv %s %s", tmp, content, tmp, path)
	if err := runSimple(client, write); err == nil {
		return nil
	}
	writeSudo := fmt.Sprintf("cat <<'EOF' | sudo -n tee %s >/dev/null\n%sEOF\nsudo -n mv %s %s", tmp, content, tmp, path)
	if err := runSimple(client, writeSudo); err != nil {
		return fmt.Errorf("write route for service %s: %w", serviceID, err)
	}
	return nil
}

func monitoringRouteConfig(serviceID string, domains []string, containerName string, containerPort int) string {
	rules := make([]string, len(domains))
	for index, domain := range domains {
		rules[index] = fmt.Sprintf("Host(`%s`)", domain)
	}
	routeName := "service-" + serviceID
	middlewareName := routeName + "-rate-limit"
	return fmt.Sprintf(`http:
  routers:
    %s:
      rule: %s
      priority: 1000
      entryPoints:
        - https
      middlewares:
        - %s
      tls:
        certResolver: letsencrypt
      service: %s
  middlewares:
    %s:
      rateLimit:
        average: 30
        burst: 60
        period: 1s
  services:
    %s:
      loadBalancer:
        servers:
          - url: %s
`, routeName, strconv.Quote(strings.Join(rules, " || ")), middlewareName, routeName, middlewareName, routeName,
		strconv.Quote(fmt.Sprintf("http://%s:%d", containerName, containerPort)))
}

func routeConfig(serviceID string, domains []string, containerName string, containerPort int) string {
	rules := make([]string, len(domains))
	for i, domain := range domains {
		rules[i] = fmt.Sprintf("Host(`%s`)", domain)
	}
	routeName := "service-" + serviceID
	return fmt.Sprintf(`http:
  routers:
    %s:
      rule: %s
      priority: 1000
      entryPoints:
        - https
      tls:
        certResolver: letsencrypt
      service: %s
  services:
    %s:
      loadBalancer:
        servers:
          - url: %s
`, routeName, strconv.Quote(strings.Join(rules, " || ")), routeName, routeName,
		strconv.Quote(fmt.Sprintf("http://%s:%d", containerName, containerPort)))
}

func RemoveRoute(client *ssh.Client, serviceID string) error {
	path := fmt.Sprintf("%s/dynamic/%s.yaml", proxyBaseDir, serviceID)
	return runElevated(client, "rm -f "+path)
}

func WaitForRoute(ctx context.Context, client *ssh.Client, serviceID, containerName string, containerPort int) error {
	want := fmt.Sprintf("http://%s:%d", containerName, containerPort)
	return waitForRoute(ctx, client, serviceID, want)
}

func WaitForRouteRemoved(ctx context.Context, client *ssh.Client, serviceID string) error {
	return waitForRoute(ctx, client, serviceID, "")
}

func waitForRoute(ctx context.Context, client *ssh.Client, serviceID, want string) error {
	deadlineCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	path := "/api/http/services/" + url.PathEscape("service-"+serviceID+"@file")

	for {
		status, body, err := traefikAPIRequest(deadlineCtx, client, path)
		if want == "" {
			if err == nil && status == http.StatusNotFound {
				return nil
			}
		} else if err == nil && status == http.StatusOK && routeBackend(body) == want {
			return nil
		}

		select {
		case <-deadlineCtx.Done():
			if err != nil {
				return fmt.Errorf("wait for route %s: %w", serviceID, err)
			}
			return fmt.Errorf("route %s did not apply expected backend %q (HTTP %d)", serviceID, want, status)
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func routeBackend(body []byte) string {
	var service struct {
		LoadBalancer struct {
			Servers []struct {
				URL string `json:"url"`
			} `json:"servers"`
		} `json:"loadBalancer"`
	}
	if json.Unmarshal(body, &service) != nil || len(service.LoadBalancer.Servers) != 1 {
		return ""
	}
	return service.LoadBalancer.Servers[0].URL
}

type LegacyRoute struct {
	ContainerName string
	Domains       []string
	ContainerPort int
}

func LegacyRouteForContainer(ctx context.Context, client *ssh.Client, containerName string) (LegacyRoute, error) {
	running, err := client.Run(ctx, fmt.Sprintf("%s inspect --format '{{.State.Running}}' %s", client.DockerBin(), containerName))
	if err != nil || running != "true" {
		return LegacyRoute{}, ErrLegacyRouteNotFound
	}
	labelsJSON, err := client.Run(ctx, fmt.Sprintf("%s inspect --format '{{json .Config.Labels}}' %s", client.DockerBin(), containerName))
	if err != nil {
		return LegacyRoute{}, fmt.Errorf("inspect legacy container labels: %w", err)
	}
	var labels map[string]string
	if err := json.Unmarshal([]byte(labelsJSON), &labels); err != nil {
		return LegacyRoute{}, fmt.Errorf("decode legacy container labels: %w", err)
	}
	router := strings.ReplaceAll(containerName, ".", "-")
	rule := labels["traefik.http.routers."+router+".rule"]
	port, err := strconv.Atoi(labels["traefik.http.services."+router+".loadbalancer.server.port"])
	if rule == "" || err != nil || port < 1 {
		return LegacyRoute{}, fmt.Errorf("legacy container %s has no recoverable Traefik route", containerName)
	}
	domains := hostnamesFromRule(rule)
	if len(domains) == 0 {
		return LegacyRoute{}, fmt.Errorf("legacy container %s has no recoverable domain rule", containerName)
	}
	return LegacyRoute{ContainerName: containerName, Domains: domains, ContainerPort: port}, nil
}

func hostnamesFromRule(rule string) []string {
	const prefix = "Host(`"
	var domains []string
	for rest := rule; ; {
		start := strings.Index(rest, prefix)
		if start < 0 {
			return domains
		}
		rest = rest[start+len(prefix):]
		end := strings.Index(rest, "`)")
		if end < 0 {
			return domains
		}
		domains = append(domains, rest[:end])
		rest = rest[end+2:]
	}
}

func RouteContainer(ctx context.Context, client *ssh.Client, serviceID string) (string, error) {
	path := "/api/http/services/" + url.PathEscape("service-"+serviceID+"@file")
	status, body, err := traefikAPIRequest(ctx, client, path)
	if err != nil {
		return "", err
	}
	if status == http.StatusNotFound {
		return "", nil
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("read route %s: HTTP %d", serviceID, status)
	}
	backend := routeBackend(body)
	parsed, err := url.Parse(backend)
	if err != nil || parsed.Hostname() == "" {
		return "", fmt.Errorf("read route %s: invalid backend %q", serviceID, backend)
	}
	return parsed.Hostname(), nil
}
