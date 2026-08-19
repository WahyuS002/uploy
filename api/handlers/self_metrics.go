package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/respond"
	"github.com/WahyuS002/uploy/telemetry"
)

// Healthz reports process liveness without touching dependencies.
func (s *Server) Healthz(w http.ResponseWriter, _ *http.Request) {
	respond.JSON(w, http.StatusOK, gen.HealthResponse{Status: gen.HealthResponseStatusOk})
}

// Readyz reports whether the API can serve requests that need PostgreSQL.
func (s *Server) Readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()
	if db.Pool == nil {
		respond.JSON(w, http.StatusServiceUnavailable, gen.ReadinessResponse{Status: gen.ReadinessResponseStatusNotReady, Database: gen.ReadinessResponseDatabaseUnavailable})
		return
	}
	if err := db.Pool.Ping(ctx); err != nil {
		respond.JSON(w, http.StatusServiceUnavailable, gen.ReadinessResponse{Status: gen.ReadinessResponseStatusNotReady, Database: gen.ReadinessResponseDatabaseUnavailable})
		return
	}
	respond.JSON(w, http.StatusOK, gen.ReadinessResponse{Status: gen.ReadinessResponseStatusReady, Database: gen.ReadinessResponseDatabaseOk})
}

// Metrics exports the internal API metrics in Prometheus text format.
func (s *Server) Metrics(w http.ResponseWriter, r *http.Request) {
	telemetry.Default.Handler(w, r)
}
