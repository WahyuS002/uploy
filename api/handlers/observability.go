package handlers

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/jobs"
	"github.com/WahyuS002/uploy/monitoring"
	"github.com/WahyuS002/uploy/respond"
	"github.com/WahyuS002/uploy/ssh"
)

const observabilityRefreshSeconds = 15
const observabilityServerConcurrency = 64
const observabilitySampleMaxAge = 45 * time.Second
const dockerFieldSeparator = "|"

var dockerSizePattern = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)\s*([[:alpha:]]*)$`)

type observedService struct {
	serviceIndex int
	deploymentID string
	container    string
}

type observedHistory struct {
	serviceIndex    int
	deploymentIndex int
	deploymentID    string
}

type containerInspect struct {
	id        string
	name      string
	state     string
	startedAt time.Time
}

type containerStats struct {
	id               string
	name             string
	cpuPercent       float64
	memoryUsedBytes  int64
	memoryLimitBytes int64
	networkInBytes   int64
	networkOutBytes  int64
}

// GetProjectObservability samples active deployment containers across every project environment.
func (s *Server) GetProjectObservability(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := s.requireProject(w, r, id); !ok {
		return
	}

	ctx := r.Context()
	services, err := db.ListServicesByProject(ctx, id)
	if err != nil {
		log.Printf("ListServicesByProject project=%s error: %v", id, err)
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list project services"})
		return
	}
	environments, err := db.ListEnvironmentsByProject(ctx, id)
	if err != nil {
		log.Printf("ListEnvironmentsByProject project=%s error: %v", id, err)
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list project environments"})
		return
	}

	environmentNames := make(map[string]string, len(environments))
	for _, environment := range environments {
		environmentNames[environment.ID] = environment.Name
	}

	response := gen.ProjectObservabilityResponse{
		SampledAt:           time.Now().UTC(),
		RefreshAfterSeconds: observabilityRefreshSeconds,
		Services:            make([]gen.ServiceObservability, len(services)),
		Summary: gen.ProjectObservabilitySummary{
			TotalServices: len(services),
		},
	}
	byServer := make(map[string][]observedService)

	for serviceIndex, service := range services {
		response.Services[serviceIndex] = gen.ServiceObservability{
			ServiceId:       service.ID,
			Name:            service.Name,
			EnvironmentName: environmentNames[service.EnvironmentID],
			Status:          gen.ServiceObservabilityStatusNotDeployed,
		}
		if !service.HasDeployed {
			continue
		}

		deployment, config, configErr := db.GetActiveDeploymentConfig(ctx, service.ID)
		if configErr != nil {
			setObservabilityError(&response.Services[serviceIndex], "Active deployment is unavailable")
			log.Printf("GetActiveDeploymentConfig service=%s error: %v", service.ID, configErr)
			continue
		}
		response.Services[serviceIndex].DeploymentId = stringPointer(deployment.ID)
		byServer[config.ServerID] = append(byServer[config.ServerID], observedService{
			serviceIndex: serviceIndex,
			deploymentID: deployment.ID,
			container:    jobs.ContainerNameForDeployment(config, deployment.ID),
		})
	}

	var waitGroup sync.WaitGroup
	semaphore := make(chan struct{}, observabilityServerConcurrency)
	for serverID, observedServices := range byServer {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			s.collectServerObservability(ctx, serverID, observedServices, response.Services)
		}()
	}
	waitGroup.Wait()

	for _, service := range response.Services {
		if service.Status == gen.ServiceObservabilityStatusRunning && service.Container != nil {
			response.Summary.RunningServices++
			response.Summary.CpuPercent += service.Container.CpuPercent
			response.Summary.MemoryUsedBytes += service.Container.MemoryUsedBytes
			response.Summary.MemoryLimitBytes += service.Container.MemoryLimitBytes
			response.Summary.NetworkInBytesTotal += service.Container.NetworkInBytesTotal
			response.Summary.NetworkOutBytesTotal += service.Container.NetworkOutBytesTotal
			continue
		}
		if service.Status != gen.ServiceObservabilityStatusNotDeployed {
			response.Summary.DegradedServices++
		}
	}

	respond.JSON(w, http.StatusOK, response)
}

func (s *Server) collectServerObservability(ctx context.Context, serverID string, observedServices []observedService, services []gen.ServiceObservability) {
	server, err := db.GetServerWithKey(ctx, serverID)
	if err != nil {
		log.Printf("GetServerWithKey server=%s error: %v", serverID, err)
		setServerObservabilityStatus(observedServices, services, gen.ServiceObservabilityStatusError, "Deployment server is unavailable")
		return
	}
	if server.Monitoring.Enabled {
		s.collectAgentObservability(ctx, server, observedServices, services)
		return
	}
	client, err := ssh.NewClient(ssh.ServerConfig{
		Host:       server.Host,
		Port:       int(server.Port),
		User:       server.SSHUser,
		PrivateKey: server.PrivateKey,
	})
	if err != nil {
		log.Printf("observability SSH connect server=%s error: %v", serverID, err)
		setServerObservabilityStatus(observedServices, services, gen.ServiceObservabilityStatusUnreachable, "Could not reach deployment server")
		return
	}
	defer client.Close()

	if err := client.DetectDocker(); err != nil {
		log.Printf("observability Docker access server=%s error: %v", serverID, err)
		setServerObservabilityStatus(observedServices, services, gen.ServiceObservabilityStatusUnreachable, "Could not reach Docker on deployment server")
		return
	}

	containerNames := make([]string, len(observedServices))
	for index, observedService := range observedServices {
		containerNames[index] = observedService.container
	}
	inspectOutput, err := client.Run(ctx, dockerInspectCommand(client.DockerBin(), containerNames))
	if err != nil {
		log.Printf("observability inspect server=%s error: %v", serverID, err)
		setServerObservabilityStatus(observedServices, services, gen.ServiceObservabilityStatusError, "Could not inspect deployment containers")
		return
	}
	inspections, err := parseDockerInspect(inspectOutput)
	if err != nil {
		log.Printf("observability inspect parse server=%s error: %v", serverID, err)
		setServerObservabilityStatus(observedServices, services, gen.ServiceObservabilityStatusError, "Could not read deployment container state")
		return
	}

	runningNames := make([]string, 0, len(observedServices))
	for _, observedService := range observedServices {
		inspection, found := inspections[observedService.container]
		if !found || inspection.state == "missing" {
			services[observedService.serviceIndex].Status = gen.ServiceObservabilityStatusStopped
			continue
		}
		container := containerFromInspect(inspection)
		services[observedService.serviceIndex].Container = &container
		if inspection.state != "running" {
			services[observedService.serviceIndex].Status = gen.ServiceObservabilityStatusStopped
			continue
		}
		runningNames = append(runningNames, observedService.container)
	}
	if len(runningNames) == 0 {
		return
	}

	statsOutput, err := client.Run(ctx, dockerStatsCommand(client.DockerBin(), runningNames))
	if err != nil {
		log.Printf("observability stats server=%s error: %v", serverID, err)
		setRunningStatsError(observedServices, services, "Could not read deployment container metrics")
		return
	}
	stats, err := parseDockerStats(statsOutput)
	if err != nil {
		log.Printf("observability stats parse server=%s error: %v", serverID, err)
		setRunningStatsError(observedServices, services, "Could not read deployment container metrics")
		return
	}
	for _, observedService := range observedServices {
		service := &services[observedService.serviceIndex]
		if service.Container == nil || service.Container.State != "running" {
			continue
		}
		stat, found := stats[observedService.container]
		if !found {
			setObservabilityError(service, "Container metrics are unavailable")
			continue
		}
		service.Status = gen.ServiceObservabilityStatusRunning
		service.Container.CpuPercent = stat.cpuPercent
		service.Container.MemoryUsedBytes = stat.memoryUsedBytes
		service.Container.MemoryLimitBytes = stat.memoryLimitBytes
		service.Container.NetworkInBytesTotal = stat.networkInBytes
		service.Container.NetworkOutBytesTotal = stat.networkOutBytes
	}
}

func (s *Server) collectAgentObservability(ctx context.Context, server db.ServerWithKey, observedServices []observedService, services []gen.ServiceObservability) {
	latest, err := monitoring.GetLatestAll(ctx, monitoring.PrivateURL(server.Monitoring.PrivateAddress, int(server.Monitoring.Port)), server.ControlToken)
	if err != nil {
		log.Printf("observability agent server=%s error: %v", server.ID, err)
		setServerObservabilityStatus(observedServices, services, gen.ServiceObservabilityStatusUnreachable, "Could not reach monitoring agent")
		return
	}
	metrics := make(map[string]monitoring.HistoryPoint, len(latest.Points))
	for _, metric := range latest.Points {
		metrics[metric.DeploymentID] = metric
	}
	for _, observedService := range observedServices {
		service := &services[observedService.serviceIndex]
		metric, found := metrics[observedService.deploymentID]
		if !found {
			service.Status = gen.ServiceObservabilityStatusStopped
			continue
		}
		if time.Since(time.UnixMilli(metric.SampledAt)) > observabilitySampleMaxAge {
			setObservabilityError(service, "Monitoring sample is stale")
			continue
		}
		container := containerFromMetric(metric)
		service.Container = &container
		if metric.State != "running" {
			service.Status = gen.ServiceObservabilityStatusStopped
			continue
		}
		service.Status = gen.ServiceObservabilityStatusRunning
	}
}

func containerFromMetric(metric monitoring.HistoryPoint) gen.ContainerObservability {
	return gen.ContainerObservability{
		Id: metric.ContainerID, Name: metric.ContainerName, State: metric.State, UptimeSeconds: metric.UptimeSeconds,
		CpuPercent: metric.CPUPercent, MemoryUsedBytes: metric.MemoryUsedBytes, MemoryLimitBytes: metric.MemoryLimitBytes,
		NetworkInBytesTotal: metric.NetworkInBytesTotal, NetworkOutBytesTotal: metric.NetworkOutBytesTotal,
	}
}

func (s *Server) GetProjectObservabilityHistory(w http.ResponseWriter, r *http.Request, id string, params gen.GetProjectObservabilityHistoryParams) {
	if _, ok := s.requireProject(w, r, id); !ok {
		return
	}
	from, err := observabilityHistorySince(params.Since)
	if err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: err.Error()})
		return
	}
	maxPoints := 300
	if params.MaxPoints != nil {
		maxPoints = *params.MaxPoints
	}
	if maxPoints < 1 || maxPoints > 1000 {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "max_points must be between 1 and 1000"})
		return
	}
	services, err := db.ListServicesByProject(r.Context(), id)
	if err != nil {
		log.Printf("ListServicesByProject project=%s error: %v", id, err)
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list project services"})
		return
	}
	response := gen.ProjectObservabilityHistoryResponse{Services: make([]gen.ServiceObservabilityHistory, len(services))}
	byServer := make(map[string][]observedHistory)
	for serviceIndex, service := range services {
		response.Services[serviceIndex] = gen.ServiceObservabilityHistory{ServiceId: service.ID, Name: service.Name, Deployments: []gen.DeploymentObservabilityHistory{}}
		deployments, err := db.ListDeploymentsByService(r.Context(), service.ID, 100)
		if err != nil {
			log.Printf("ListDeploymentsByService service=%s error: %v", service.ID, err)
			continue
		}
		for _, deployment := range deployments {
			if deployment.Status != "success" {
				continue
			}
			deploymentIndex := len(response.Services[serviceIndex].Deployments)
			response.Services[serviceIndex].Deployments = append(response.Services[serviceIndex].Deployments, gen.DeploymentObservabilityHistory{
				DeploymentId: deployment.ID, Points: []gen.ContainerObservabilitySample{},
			})
			deploymentConfig, err := db.GetDeploymentConfig(r.Context(), deployment.ID)
			if err != nil {
				response.Services[serviceIndex].Deployments[deploymentIndex].UnavailableReason = stringPointer("Deployment configuration is unavailable")
				continue
			}
			byServer[deploymentConfig.ServerID] = append(byServer[deploymentConfig.ServerID], observedHistory{
				serviceIndex: serviceIndex, deploymentIndex: deploymentIndex, deploymentID: deployment.ID,
			})
		}
	}
	to := time.Now().UTC()
	for serverID, observations := range byServer {
		s.collectServerHistory(r.Context(), serverID, observations, response.Services, from, to, maxPoints)
	}
	respond.JSON(w, http.StatusOK, response)
}

func (s *Server) collectServerHistory(ctx context.Context, serverID string, observations []observedHistory, services []gen.ServiceObservabilityHistory, from, to time.Time, maxPoints int) {
	server, err := db.GetServerWithKey(ctx, serverID)
	if err != nil {
		setHistoryUnavailable(observations, services, "Deployment server is unavailable")
		return
	}
	if !server.Monitoring.Enabled {
		setHistoryUnavailable(observations, services, "Retained monitoring is not enabled on this server")
		return
	}
	ids := make([]string, len(observations))
	for index, observation := range observations {
		ids[index] = observation.deploymentID
	}
	pointsPerDeployment := maxPoints / len(observations)
	if pointsPerDeployment < 1 {
		pointsPerDeployment = 1
	}
	histories, err := monitoring.GetHistories(ctx, monitoring.PrivateURL(server.Monitoring.PrivateAddress, int(server.Monitoring.Port)), server.ControlToken, ids, from, to, pointsPerDeployment)
	if err != nil {
		log.Printf("observability history agent server=%s error: %v", server.ID, err)
		setHistoryUnavailable(observations, services, "Could not reach monitoring agent")
		return
	}
	for _, observation := range observations {
		points := histories.Deployments[observation.deploymentID].Points
		mapped := make([]gen.ContainerObservabilitySample, len(points))
		for index, point := range points {
			mapped[index] = gen.ContainerObservabilitySample{
				ContainerId: point.ContainerID, ContainerName: point.ContainerName, State: point.State,
				SampledAt: time.UnixMilli(point.SampledAt).UTC(), CpuPercent: point.CPUPercent,
				MemoryUsedBytes: point.MemoryUsedBytes, MemoryLimitBytes: point.MemoryLimitBytes,
				NetworkInBytesTotal: point.NetworkInBytesTotal, NetworkOutBytesTotal: point.NetworkOutBytesTotal,
				UptimeSeconds: point.UptimeSeconds,
			}
		}
		services[observation.serviceIndex].Deployments[observation.deploymentIndex].Points = mapped
	}
}

func setHistoryUnavailable(observations []observedHistory, services []gen.ServiceObservabilityHistory, reason string) {
	for _, observation := range observations {
		services[observation.serviceIndex].Deployments[observation.deploymentIndex].UnavailableReason = stringPointer(reason)
	}
}

func observabilityHistorySince(value *gen.GetProjectObservabilityHistoryParamsSince) (time.Time, error) {
	duration := 7 * 24 * time.Hour
	if value != nil {
		switch string(*value) {
		case "1h":
			duration = time.Hour
		case "6h":
			duration = 6 * time.Hour
		case "24h":
			duration = 24 * time.Hour
		case "7d":
			duration = 7 * 24 * time.Hour
		case "30d":
			duration = 30 * 24 * time.Hour
		default:
			return time.Time{}, fmt.Errorf("since must be one of 1h, 6h, 24h, 7d, or 30d")
		}
	}
	return time.Now().UTC().Add(-duration), nil
}

func setServerObservabilityStatus(observedServices []observedService, services []gen.ServiceObservability, status gen.ServiceObservabilityStatus, message string) {
	for _, observedService := range observedServices {
		services[observedService.serviceIndex].Status = status
		services[observedService.serviceIndex].Error = stringPointer(message)
	}
}

func setRunningStatsError(observedServices []observedService, services []gen.ServiceObservability, message string) {
	for _, observedService := range observedServices {
		service := &services[observedService.serviceIndex]
		if service.Container != nil && service.Container.State == "running" {
			setObservabilityError(service, message)
		}
	}
}

func setObservabilityError(service *gen.ServiceObservability, message string) {
	service.Status = gen.ServiceObservabilityStatusError
	service.Error = stringPointer(message)
}

func containerFromInspect(inspect containerInspect) gen.ContainerObservability {
	uptime := int64(0)
	if !inspect.startedAt.IsZero() {
		uptime = int64(time.Since(inspect.startedAt).Seconds())
		if uptime < 0 {
			uptime = 0
		}
	}
	return gen.ContainerObservability{
		Id:            inspect.id,
		Name:          inspect.name,
		State:         inspect.state,
		UptimeSeconds: uptime,
	}
}

func dockerInspectCommand(dockerBin string, containers []string) string {
	quotedContainers := shellQuoteAll(containers)
	return fmt.Sprintf(`for container in %s; do %s inspect --format '{{.ID}}|{{.Name}}|{{.State.Status}}|{{.State.StartedAt}}' "$container" 2>/dev/null || printf '|%%s|missing|\n' "$container"; done`, quotedContainers, dockerBin)
}

func dockerStatsCommand(dockerBin string, containers []string) string {
	return fmt.Sprintf(`%s stats --no-stream --format '{{.ID}}|{{.Name}}|{{.CPUPerc}}|{{.MemUsage}}|{{.NetIO}}' %s`, dockerBin, shellQuoteAll(containers))
}

func shellQuoteAll(values []string) string {
	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = ssh.ShellQuote(value)
	}
	return strings.Join(quoted, " ")
}

func parseDockerInspect(output string) (map[string]containerInspect, error) {
	inspections := make(map[string]containerInspect)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), dockerFieldSeparator)
		if len(fields) != 4 {
			return nil, fmt.Errorf("invalid docker inspect row %q", scanner.Text())
		}
		name := normalizeContainerName(fields[1])
		if name == "" {
			return nil, fmt.Errorf("docker inspect row has no container name")
		}
		startedAt := time.Time{}
		if fields[3] != "" && fields[3] != "0001-01-01T00:00:00Z" {
			parsedStartedAt, err := time.Parse(time.RFC3339Nano, fields[3])
			if err != nil {
				return nil, fmt.Errorf("parse started at for %s: %w", name, err)
			}
			startedAt = parsedStartedAt
		}
		inspections[name] = containerInspect{
			id:        fields[0],
			name:      name,
			state:     fields[2],
			startedAt: startedAt,
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return inspections, nil
}

func parseDockerStats(output string) (map[string]containerStats, error) {
	stats := make(map[string]containerStats)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), dockerFieldSeparator)
		if len(fields) != 5 {
			return nil, fmt.Errorf("invalid docker stats row %q", scanner.Text())
		}
		cpuPercent, err := parseDockerPercent(fields[2])
		if err != nil {
			return nil, err
		}
		memoryUsedBytes, memoryLimitBytes, err := parseDockerBytePair(fields[3])
		if err != nil {
			return nil, err
		}
		networkInBytes, networkOutBytes, err := parseDockerBytePair(fields[4])
		if err != nil {
			return nil, err
		}
		name := normalizeContainerName(fields[1])
		if name == "" {
			return nil, fmt.Errorf("docker stats row has no container name")
		}
		stats[name] = containerStats{
			id:               fields[0],
			name:             name,
			cpuPercent:       cpuPercent,
			memoryUsedBytes:  memoryUsedBytes,
			memoryLimitBytes: memoryLimitBytes,
			networkInBytes:   networkInBytes,
			networkOutBytes:  networkOutBytes,
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return stats, nil
}

func parseDockerPercent(value string) (float64, error) {
	percent, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(value), "%"), 64)
	if err != nil {
		return 0, fmt.Errorf("parse Docker CPU percent %q: %w", value, err)
	}
	return percent, nil
}

func parseDockerBytePair(value string) (int64, int64, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid Docker byte pair %q", value)
	}
	first, err := parseDockerBytes(parts[0])
	if err != nil {
		return 0, 0, err
	}
	second, err := parseDockerBytes(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return first, second, nil
}

func parseDockerBytes(value string) (int64, error) {
	matches := dockerSizePattern.FindStringSubmatch(strings.TrimSpace(value))
	if matches == nil {
		return 0, fmt.Errorf("invalid Docker byte value %q", value)
	}
	amount, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("parse Docker byte value %q: %w", value, err)
	}
	multiplier, ok := dockerByteMultiplier[strings.ToLower(matches[2])]
	if !ok {
		return 0, fmt.Errorf("unknown Docker byte unit %q", matches[2])
	}
	return int64(amount * multiplier), nil
}

var dockerByteMultiplier = map[string]float64{
	"":    1,
	"b":   1,
	"kb":  1_000,
	"kib": 1 << 10,
	"mb":  1_000_000,
	"mib": 1 << 20,
	"gb":  1_000_000_000,
	"gib": 1 << 30,
	"tb":  1_000_000_000_000,
	"tib": 1 << 40,
}

func normalizeContainerName(name string) string {
	return strings.TrimPrefix(strings.TrimSpace(name), "/")
}

func stringPointer(value string) *string {
	return &value
}
