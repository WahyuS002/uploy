package handlers

import (
	"context"
	"fmt"
	"github.com/WahyuS002/uploy/telemetry"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/jobs"
	"github.com/WahyuS002/uploy/monitoring"
	"github.com/WahyuS002/uploy/respond"
)

const observabilityRefreshSeconds = 15
const observabilityServerConcurrency = 64
const observabilitySampleMaxAge = 45 * time.Second

type serverObservationTarget struct {
	serviceIndex int
	deploymentID string
	container    string
}

type observedHistory struct {
	serviceIndex    int
	deploymentIndex int
	deploymentID    string
}

// GetProjectObservability samples active deployment containers across every project environment.
func (s *Server) GetProjectObservability(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := s.requireProject(w, r, id); !ok {
		return
	}

	ctx := r.Context()
	services, err := db.ListServicesByProject(ctx, id)
	if err != nil {
		telemetry.Printf("ListServicesByProject project=%s error: %v", id, err)
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list project services"})
		return
	}
	environments, err := db.ListEnvironmentsByProject(ctx, id)
	if err != nil {
		telemetry.Printf("ListEnvironmentsByProject project=%s error: %v", id, err)
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
	targetsByServer := make(map[string][]serverObservationTarget)

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
			telemetry.Printf("GetActiveDeploymentConfig service=%s error: %v", service.ID, configErr)
			continue
		}
		response.Services[serviceIndex].DeploymentId = stringPointer(deployment.ID)
		targetsByServer[config.ServerID] = append(targetsByServer[config.ServerID], serverObservationTarget{
			serviceIndex: serviceIndex,
			deploymentID: deployment.ID,
			container:    jobs.ContainerNameForDeployment(config, deployment.ID),
		})
	}

	// Indexed rather than appended under a lock: each goroutine owns one slot, and
	// the map's iteration order would make the response order jitter between polls.
	serverIDs := make([]string, 0, len(targetsByServer))
	for serverID := range targetsByServer {
		serverIDs = append(serverIDs, serverID)
	}
	unmonitored := make([]*gen.UnmonitoredServer, len(serverIDs))
	var waitGroup sync.WaitGroup
	semaphore := make(chan struct{}, observabilityServerConcurrency)
	for index, serverID := range serverIDs {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			unmonitored[index] = s.collectServerObservability(ctx, serverID, targetsByServer[serverID], response.Services)
		}()
	}
	waitGroup.Wait()

	response.UnmonitoredServers = []gen.UnmonitoredServer{}
	for _, server := range unmonitored {
		if server != nil {
			response.UnmonitoredServers = append(response.UnmonitoredServers, *server)
		}
	}
	sort.Slice(response.UnmonitoredServers, func(left, right int) bool {
		return response.UnmonitoredServers[left].Name < response.UnmonitoredServers[right].Name
	})

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
		// A server without an agent is unconfigured, not degraded: counting it here
		// would light up "needs attention" for a project that is perfectly healthy.
		if service.Status != gen.ServiceObservabilityStatusNotDeployed &&
			service.Status != gen.ServiceObservabilityStatusMonitoringDisabled {
			response.Summary.DegradedServices++
		}
	}

	respond.JSON(w, http.StatusOK, response)
}

// collectServerObservability returns the server when it carries no agent, so the
// caller can offer one call to action per server instead of one per service.
func (s *Server) collectServerObservability(ctx context.Context, serverID string, targets []serverObservationTarget, services []gen.ServiceObservability) *gen.UnmonitoredServer {
	server, err := db.GetServerWithKey(ctx, serverID)
	if err != nil {
		telemetry.Printf("GetServerWithKey server=%s error: %v", serverID, err)
		setServerObservabilityStatus(targets, services, gen.ServiceObservabilityStatusError, "Deployment server is unavailable")
		return nil
	}
	for _, target := range targets {
		services[target.serviceIndex].ServerId = stringPointer(server.ID)
		services[target.serviceIndex].ServerName = stringPointer(server.Name)
	}
	// No agent, no metrics. Reaching for docker stats over SSH here used to fill the
	// live cards while the retained charts stayed empty, which read as a broken page
	// rather than an unconfigured one.
	if !server.Monitoring.Enabled {
		for _, target := range targets {
			services[target.serviceIndex].Status = gen.ServiceObservabilityStatusMonitoringDisabled
		}
		// One target per service, so the target count is the affected service count.
		return &gen.UnmonitoredServer{Id: server.ID, Name: server.Name, ServiceCount: len(targets)}
	}
	s.collectAgentObservability(ctx, server, targets, services)
	return nil
}

func (s *Server) collectAgentObservability(ctx context.Context, server db.ServerWithKey, targets []serverObservationTarget, services []gen.ServiceObservability) {
	latest, err := monitoring.GetLatestAll(ctx, monitoringTarget(server), server.ControlToken)
	if err != nil {
		telemetry.Printf("observability agent server=%s error: %v", server.ID, err)
		setServerObservabilityStatus(targets, services, gen.ServiceObservabilityStatusUnreachable, "Could not reach monitoring agent")
		return
	}
	metrics := make(map[string]monitoring.HistoryPoint, len(latest.Points))
	for _, metric := range latest.Points {
		metrics[metric.DeploymentID] = metric
	}
	for _, target := range targets {
		service := &services[target.serviceIndex]
		metric, found := metrics[target.deploymentID]
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
		telemetry.Printf("ListServicesByProject project=%s error: %v", id, err)
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list project services"})
		return
	}
	response := gen.ProjectObservabilityHistoryResponse{
		Services:          make([]gen.ServiceObservabilityHistory, len(services)),
		DeploymentMarkers: []gen.DeploymentMarker{},
	}
	byServer := make(map[string][]observedHistory)
	to := time.Now().UTC()
	for serviceIndex, service := range services {
		response.Services[serviceIndex] = gen.ServiceObservabilityHistory{ServiceId: service.ID, Name: service.Name, Deployments: []gen.DeploymentObservabilityHistory{}}
		deployments, err := db.ListDeploymentsByService(r.Context(), service.ID, 100)
		if err != nil {
			telemetry.Printf("ListDeploymentsByService service=%s error: %v", service.ID, err)
			continue
		}
		for _, deployment := range deployments {
			if deployment.CreatedAt.Before(from) || deployment.CreatedAt.After(to) {
				continue
			}
			deploymentConfig, configErr := db.GetDeploymentConfig(r.Context(), deployment.ID)
			image := ""
			if configErr == nil {
				image = deploymentConfig.Image
			}
			response.DeploymentMarkers = append(response.DeploymentMarkers, deploymentMarker(service, deployment, image, to))
			if deployment.Status != "success" || configErr != nil {
				continue
			}
			deploymentIndex := len(response.Services[serviceIndex].Deployments)
			response.Services[serviceIndex].Deployments = append(response.Services[serviceIndex].Deployments, gen.DeploymentObservabilityHistory{
				DeploymentId: deployment.ID, Points: []gen.ContainerObservabilitySample{},
			})
			byServer[deploymentConfig.ServerID] = append(byServer[deploymentConfig.ServerID], observedHistory{
				serviceIndex: serviceIndex, deploymentIndex: deploymentIndex, deploymentID: deployment.ID,
			})
		}
	}
	for serverID, observations := range byServer {
		s.collectServerHistory(r.Context(), serverID, observations, response.Services, from, to, maxPoints)
	}
	respond.JSON(w, http.StatusOK, response)
}

func deploymentMarker(service db.Service, deployment db.Deployment, image string, now time.Time) gen.DeploymentMarker {
	start := deployment.StartedAt
	if start.IsZero() {
		start = deployment.CreatedAt
	}
	end := now
	if deployment.CompletedAt != nil && !deployment.CompletedAt.IsZero() && deployment.Status != "in_progress" {
		end = *deployment.CompletedAt
	}
	duration := int(end.Sub(start).Seconds())
	if duration < 0 {
		duration = 0
	}
	return gen.DeploymentMarker{
		DeploymentId:    deployment.ID,
		ServiceId:       service.ID,
		ServiceName:     service.Name,
		At:              deployment.CreatedAt,
		Status:          gen.DeploymentMarkerStatus(deployment.Status),
		DurationSeconds: duration,
		Image:           image,
		Commit:          imageCommit(image),
	}
}

func imageCommit(image string) string {
	if digestIndex := strings.LastIndex(image, "@sha256:"); digestIndex >= 0 {
		digest := image[digestIndex+len("@sha256:"):]
		if len(digest) > 12 {
			return digest[:12]
		}
		return digest
	}
	last := image[strings.LastIndex(image, "/")+1:]
	colon := strings.LastIndex(last, ":")
	if colon < 0 || colon == len(last)-1 {
		return ""
	}
	tag := last[colon+1:]
	if len(tag) < 7 || len(tag) > 40 {
		return ""
	}
	for _, char := range tag {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return ""
		}
	}
	return tag
}

func (s *Server) collectServerHistory(ctx context.Context, serverID string, observations []observedHistory, services []gen.ServiceObservabilityHistory, from, to time.Time, maxPoints int) {
	server, err := db.GetServerWithKey(ctx, serverID)
	if err != nil {
		telemetry.Printf("observability history server=%s error: %v", serverID, err)
		return
	}
	// The live snapshot already reports monitoring_disabled for these services, so the
	// chart simply has nothing to draw.
	if !server.Monitoring.Enabled {
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
	histories, err := monitoring.GetHistories(ctx, monitoringTarget(server), server.ControlToken, ids, from, to, pointsPerDeployment)
	if err != nil {
		telemetry.Printf("observability history agent server=%s error: %v", server.ID, err)
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

func setServerObservabilityStatus(targets []serverObservationTarget, services []gen.ServiceObservability, status gen.ServiceObservabilityStatus, message string) {
	for _, target := range targets {
		services[target.serviceIndex].Status = status
		services[target.serviceIndex].Error = stringPointer(message)
	}
}

func setObservabilityError(service *gen.ServiceObservability, message string) {
	service.Status = gen.ServiceObservabilityStatusError
	service.Error = stringPointer(message)
}

func stringPointer(value string) *string {
	return &value
}
