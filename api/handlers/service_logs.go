package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/WahyuS002/uploy/auth"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/jobs"
	"github.com/WahyuS002/uploy/respond"
	"github.com/WahyuS002/uploy/ssh"
	"github.com/jackc/pgx/v5"
)

// GetServiceLogs streams a running container's stdout and stderr over SSE.
func (s *Server) GetServiceLogs(w http.ResponseWriter, r *http.Request, id string, params gen.GetServiceLogsParams) {
	sc, _ := auth.GetSessionContext(r)

	// Checked up here, before anything opens an SSH session on its behalf and
	// before the response becomes an event stream, where a bad value could only
	// be reported as an in-band event.
	sinceFlag, ok := logSinceFlag((*string)(params.Since))
	if !ok {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "unknown time range: " + string(*params.Since)})
		return
	}

	svc, err := db.GetServiceWithServer(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "service not found"})
		} else {
			log.Printf("GetServiceWithServer id=%s error: %v", id, err)
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to look up service"})
		}
		return
	}

	// Same as everywhere else: 404 rather than 403 across workspaces, so the
	// response does not confirm that the service exists.
	if svc.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "service not found"})
		return
	}

	if !svc.HasDeployed {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "service has not been deployed yet, so it has no logs"})
		return
	}
	activeDeployment, activeConfig, err := db.GetActiveDeploymentConfig(r.Context(), svc.ID)
	if err != nil {
		log.Printf("GetActiveDeploymentConfig service=%s error: %v", id, err)
		respond.JSON(w, http.StatusConflict, gen.ErrorResponse{Error: "service has no active deployment"})
		return
	}

	streamContainerLogs(w, r, ssh.ServerConfig{
		Host:       svc.Host,
		Port:       int(svc.ServerPort),
		User:       svc.SSHUser,
		PrivateKey: svc.PrivateKey,
	}, jobs.ContainerNameForDeployment(activeConfig, activeDeployment.ID), sinceFlag, "service logs id="+id)
}
