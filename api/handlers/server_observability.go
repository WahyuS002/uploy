package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/WahyuS002/uploy/auth"
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/monitoring"
	"github.com/WahyuS002/uploy/respond"
)

func (s *Server) GetServerObservability(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)
	server, ok := s.workspaceServerWithKey(w, r, id, sc.WorkspaceID)
	if !ok {
		return
	}
	if !server.Monitoring.Enabled {
		respond.JSON(w, http.StatusConflict, gen.ErrorResponse{Error: "monitoring is not enabled on this server"})
		return
	}
	latest, err := monitoring.GetServerLatest(r.Context(), monitoringTarget(server), server.ControlToken)
	if err != nil {
		respond.JSON(w, http.StatusBadGateway, gen.ErrorResponse{Error: "could not reach monitoring agent"})
		return
	}
	respond.JSON(w, http.StatusOK, serverObservabilityResponse(latest))
}

func (s *Server) GetServerObservabilityHistory(w http.ResponseWriter, r *http.Request, id string, params gen.GetServerObservabilityHistoryParams) {
	sc, _ := auth.GetSessionContext(r)
	server, ok := s.workspaceServerWithKey(w, r, id, sc.WorkspaceID)
	if !ok {
		return
	}
	if !server.Monitoring.Enabled {
		respond.JSON(w, http.StatusConflict, gen.ErrorResponse{Error: "monitoring is not enabled on this server"})
		return
	}
	from, err := serverObservabilityHistorySince(params.Since)
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
	to := time.Now().UTC()
	history, err := monitoring.GetServerHistory(r.Context(), monitoringTarget(server), server.ControlToken, from, to, maxPoints)
	if err != nil {
		respond.JSON(w, http.StatusBadGateway, gen.ErrorResponse{Error: "could not reach monitoring agent"})
		return
	}
	points := make([]gen.ServerObservabilityResponse, len(history.Points))
	for index, point := range history.Points {
		points[index] = serverObservabilityResponse(point)
	}
	respond.JSON(w, http.StatusOK, gen.ServerObservabilityHistoryResponse{Points: points})
}

func serverObservabilityResponse(point monitoring.ServerLatestResponse) gen.ServerObservabilityResponse {
	partitions := make([]gen.ServerDiskPartition, len(point.Partitions))
	for index, partition := range point.Partitions {
		partitions[index] = gen.ServerDiskPartition{
			Mountpoint:  partition.Mountpoint,
			UsedBytes:   partition.UsedBytes,
			TotalBytes:  partition.TotalBytes,
			UsedPercent: partition.UsedPercent,
		}
	}
	return gen.ServerObservabilityResponse{
		SampledAt:           time.UnixMilli(point.SampledAt).UTC(),
		DiskUsedBytes:       point.DiskUsedBytes,
		DiskTotalBytes:      point.DiskTotalBytes,
		DiskUsedPercent:     point.DiskUsedPercent,
		DiskReadBytesTotal:  point.DiskReadBytesTotal,
		DiskWriteBytesTotal: point.DiskWriteBytesTotal,
		Load1:               point.Load1,
		Load5:               point.Load5,
		Load15:              point.Load15,
		SwapUsedBytes:       point.SwapUsedBytes,
		SwapTotalBytes:      point.SwapTotalBytes,
		Partitions:          partitions,
	}
}

func serverObservabilityHistorySince(value *gen.GetServerObservabilityHistoryParamsSince) (time.Time, error) {
	if value == nil {
		return time.Now().UTC().Add(-7 * 24 * time.Hour), nil
	}
	switch string(*value) {
	case "1h":
		return time.Now().UTC().Add(-time.Hour), nil
	case "6h":
		return time.Now().UTC().Add(-6 * time.Hour), nil
	case "24h":
		return time.Now().UTC().Add(-24 * time.Hour), nil
	case "7d":
		return time.Now().UTC().Add(-7 * 24 * time.Hour), nil
	case "30d":
		return time.Now().UTC().Add(-30 * 24 * time.Hour), nil
	default:
		return time.Time{}, fmt.Errorf("since must be one of 1h, 6h, 24h, 7d, or 30d")
	}
}
