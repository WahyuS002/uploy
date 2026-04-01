package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/WahyuS002/uploy/auth"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/respond"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// validContainerName matches Docker container name rules: [a-zA-Z0-9][a-zA-Z0-9_.-]*
var validContainerName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

// validImage matches Docker image references: alphanumeric with / : . -
var validImage = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_./:@-]*$`)

// validFQDN matches valid hostnames: labels separated by dots, each 1-63 chars, total <= 253
var validFQDN = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func validatePort(port int) string {
	if port < 1 || port > 65535 {
		return "port must be between 1 and 65535"
	}
	if port == 80 || port == 443 {
		return "ports 80 and 443 are reserved for the Uploy proxy; use another port for direct access"
	}
	return ""
}

func (s *Server) CreateService(w http.ResponseWriter, r *http.Request) {
	sc, _ := auth.GetSessionContext(r)

	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	var req gen.CreateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "name is required"})
		return
	}
	req.Image = strings.TrimSpace(req.Image)
	if req.Image == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "image is required"})
		return
	}
	req.ContainerName = strings.TrimSpace(req.ContainerName)
	if req.ContainerName == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "container_name is required"})
		return
	}
	if !validContainerName.MatchString(req.ContainerName) {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "container_name contains invalid characters"})
		return
	}
	if !validImage.MatchString(req.Image) {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "image contains invalid characters"})
		return
	}

	// Verify server exists and belongs to the same workspace
	server, err := db.GetServerByID(r.Context(), req.ServerId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "server not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to look up server"})
		}
		return
	}
	if server.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "server not found"})
		return
	}

	if msg := validatePort(req.Port); msg != "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: msg})
		return
	}

	// Validate environment ownership chain: env -> project -> workspace
	env, err := db.GetEnvironmentByID(r.Context(), req.EnvironmentId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "environment not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to look up environment"})
		}
		return
	}
	proj, err := db.GetProjectByID(r.Context(), env.ProjectID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to look up project"})
		return
	}
	if proj.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "environment not found"})
		return
	}

	kind := "application"
	if req.Kind != nil {
		if !req.Kind.Valid() {
			respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid kind"})
			return
		}
		kind = string(*req.Kind)
	}
	if kind != "application" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "only 'application' kind is currently supported"})
		return
	}

	svc, err := db.CreateService(r.Context(), req.Name, req.Image, req.ContainerName, int32(req.Port), req.ServerId, sc.WorkspaceID, kind, proj.ID, req.EnvironmentId)
	if err != nil {
		if isUniqueViolation(err) {
			respond.JSON(w, http.StatusConflict, gen.ErrorResponse{Error: "container_name already in use on this server"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to create service"})
		}
		return
	}

	respond.JSON(w, http.StatusCreated, serviceToResponse(svc))
}

func (s *Server) ListServices(w http.ResponseWriter, r *http.Request) {
	sc, _ := auth.GetSessionContext(r)

	services, err := db.ListServicesByWorkspace(r.Context(), sc.WorkspaceID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list services"})
		return
	}

	resp := make([]gen.ServiceResponse, len(services))
	for i, svc := range services {
		resp[i] = serviceToResponse(svc)
	}

	respond.JSON(w, http.StatusOK, resp)
}

func (s *Server) GetService(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)

	svc, err := db.GetServiceByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "service not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to get service"})
		}
		return
	}

	if svc.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "service not found"})
		return
	}

	respond.JSON(w, http.StatusOK, serviceToResponse(svc))
}

func (s *Server) UpdateService(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)

	existing, err := db.GetServiceByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "service not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to get service"})
		}
		return
	}
	if existing.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "service not found"})
		return
	}

	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	var req gen.UpdateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "name is required"})
		return
	}
	req.Image = strings.TrimSpace(req.Image)
	if req.Image == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "image is required"})
		return
	}
	req.ContainerName = strings.TrimSpace(req.ContainerName)
	if req.ContainerName == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "container_name is required"})
		return
	}
	if !validContainerName.MatchString(req.ContainerName) {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "container_name contains invalid characters"})
		return
	}
	if !validImage.MatchString(req.Image) {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "image contains invalid characters"})
		return
	}

	if req.ContainerName != existing.ContainerName {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "container_name cannot be changed; delete and recreate the service instead"})
		return
	}
	if req.ServerId != existing.ServerID {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "server_id cannot be changed; delete and recreate the service instead"})
		return
	}

	if msg := validatePort(req.Port); msg != "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: msg})
		return
	}

	svc, err := db.UpdateService(r.Context(), id, req.Name, req.Image, req.ContainerName, int32(req.Port), req.ServerId)
	if err != nil {
		if isUniqueViolation(err) {
			respond.JSON(w, http.StatusConflict, gen.ErrorResponse{Error: "container_name already in use on this server"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to update service"})
		}
		return
	}

	respond.JSON(w, http.StatusOK, serviceToResponse(svc))
}

func (s *Server) DeleteService(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)

	svc, err := db.GetServiceByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "service not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to get service"})
		}
		return
	}
	if svc.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "service not found"})
		return
	}

	if sc.WorkspaceRole != "owner" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	if err := db.DeleteService(r.Context(), id); err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to delete service"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func serviceToResponse(svc db.Service) gen.ServiceResponse {
	return gen.ServiceResponse{
		Id:            svc.ID,
		Name:          svc.Name,
		Image:         svc.Image,
		ContainerName: svc.ContainerName,
		Port:          int(svc.Port),
		ServerId:      svc.ServerID,
		Kind:          gen.ServiceResponseKind(svc.Kind),
		ProjectId:     svc.ProjectID,
		EnvironmentId: svc.EnvironmentID,
		CreatedAt:     svc.CreatedAt,
		UpdatedAt:     svc.UpdatedAt,
	}
}
