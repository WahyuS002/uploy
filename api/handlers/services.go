package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/WahyuS002/uploy/auth"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/jobs"
	"github.com/WahyuS002/uploy/respond"
	"github.com/WahyuS002/uploy/ssh"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// validContainerName matches Docker container name rules: [a-zA-Z0-9][a-zA-Z0-9_.-]*
var validContainerName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

// validImage matches Docker image references: alphanumeric with / : . -
var validImage = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_./:@-]*$`)

// validFQDN matches valid hostnames: labels separated by dots, each 1-63 chars, total <= 253
var validFQDN = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

// The API models an optional port as *int while the database layer uses *int32;
// this is the one place that gap is bridged.
func int32PtrFromIntPtr(v *int) *int32 {
	if v == nil {
		return nil
	}
	i := int32(*v)
	return &i
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// validatePorts checks the two ports a service has, which are not the same
// thing and do not have the same rules.
//
// containerPort is what the image listens on inside the container — a property
// of the image, so 80 is perfectly normal there (nginx, caddy, httpd all use it).
//
// hostPort is where the service is published on the machine. Nil means it is
// not published at all: other services still reach it over the uploy network,
// but nothing outside the machine can. 80 and 443 belong to the Traefik proxy,
// so nothing else may take them.
func validatePorts(containerPort int, hostPort *int) string {
	if containerPort < 1 || containerPort > 65535 {
		return "container port must be between 1 and 65535"
	}
	if hostPort == nil {
		return ""
	}
	if *hostPort < 1 || *hostPort > 65535 {
		return "host port must be between 1 and 65535"
	}
	if *hostPort == 80 || *hostPort == 443 {
		return "ports 80 and 443 are reserved for the Uploy proxy; use another host port for direct access"
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

	req.Image = strings.TrimSpace(req.Image)
	if req.Image == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "image is required"})
		return
	}
	if !validImage.MatchString(req.Image) {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "image contains invalid characters"})
		return
	}

	// Blank name or container name means "derive it from the image", the same
	// rule POST /api/projects/from-image follows. The canvas flow only asks for
	// an image, so those are the two fields it has nothing to send.
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		req.Name = serviceNameFromImage(req.Image)
	}
	req.ContainerName = strings.TrimSpace(req.ContainerName)
	if req.ContainerName == "" {
		req.ContainerName = containerNameFromImage(req.Image)
	}
	if !validContainerName.MatchString(req.ContainerName) {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "container_name contains invalid characters"})
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

	if msg := validatePorts(req.ContainerPort, req.HostPort); msg != "" {
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

	svc, err := db.CreateService(r.Context(), req.Name, req.Image, req.ContainerName, int32(req.ContainerPort), int32PtrFromIntPtr(req.HostPort), req.ServerId, sc.WorkspaceID, kind, proj.ID, req.EnvironmentId)
	if err != nil {
		if isUniqueViolation(err) {
			respond.JSON(w, http.StatusConflict, gen.ErrorResponse{Error: "container_name already in use on this server"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to create service"})
		}
		return
	}

	resp, err := serviceResponse(r.Context(), svc)
	if err != nil {
		log.Printf("pending changes for new service %s: %v", svc.ID, err)
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to read service state"})
		return
	}
	respond.JSON(w, http.StatusCreated, resp)
}

func (s *Server) ListServices(w http.ResponseWriter, r *http.Request) {
	sc, _ := auth.GetSessionContext(r)

	services, err := db.ListServicesByWorkspace(r.Context(), sc.WorkspaceID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list services"})
		return
	}

	resp, err := serviceResponses(r.Context(), services)
	if err != nil {
		log.Printf("pending changes for service list: %v", err)
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to read service state"})
		return
	}

	respond.JSON(w, http.StatusOK, resp)
}

func (s *Server) ListProjectServices(w http.ResponseWriter, r *http.Request, id string) {
	proj, ok := s.requireProject(w, r, id)
	if !ok {
		return
	}

	services, err := db.ListServicesByProject(r.Context(), proj.ID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list services"})
		return
	}

	resp, err := serviceResponses(r.Context(), services)
	if err != nil {
		log.Printf("pending changes for service list: %v", err)
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to read service state"})
		return
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

	resp, err := serviceResponse(r.Context(), svc)
	if err != nil {
		log.Printf("pending changes for service %s: %v", svc.ID, err)
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to read service state"})
		return
	}
	respond.JSON(w, http.StatusOK, resp)
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

	if msg := validatePorts(req.ContainerPort, req.HostPort); msg != "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: msg})
		return
	}

	svc, err := db.UpdateService(r.Context(), id, req.Name, req.Image, req.ContainerName, int32(req.ContainerPort), int32PtrFromIntPtr(req.HostPort), req.ServerId)
	if err != nil {
		if isUniqueViolation(err) {
			respond.JSON(w, http.StatusConflict, gen.ErrorResponse{Error: "container_name already in use on this server"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to update service"})
		}
		return
	}

	resp, err := serviceResponse(r.Context(), svc)
	if err != nil {
		log.Printf("pending changes for updated service %s: %v", svc.ID, err)
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to read service state"})
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

func (s *Server) DeleteService(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)

	svc, err := db.GetServiceWithServer(r.Context(), id)
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
	latest, err := db.ListDeploymentsByService(r.Context(), svc.ID, 1)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to read deployment state"})
		return
	}
	if len(latest) > 0 && latest[0].Status == "in_progress" {
		respond.JSON(w, http.StatusConflict, gen.ErrorResponse{Error: "cannot delete a service while deployment is in progress"})
		return
	}

	// Tear the container down before dropping the record. Fail closed: if the
	// server is unreachable we would otherwise leave a container serving traffic
	// with nothing in Uploy pointing at it, and no way to reach it from the UI.
	// A failed delete is recoverable, an orphan is not.
	if svc.HasDeployed {
		activeDeployment, activeConfig, activeErr := db.GetActiveDeploymentConfig(r.Context(), svc.ID)
		if activeErr != nil {
			respond.JSON(w, http.StatusConflict, gen.ErrorResponse{Error: "service has no active deployment"})
			return
		}
		if err := jobs.RemoveService(ssh.ServerConfig{
			Host:       svc.Host,
			Port:       int(svc.ServerPort),
			User:       svc.SSHUser,
			PrivateKey: svc.PrivateKey,
		}, svc.ID, jobs.ContainerNameForDeployment(activeConfig, activeDeployment.ID)); err != nil {
			log.Printf("RemoveContainer service=%s error: %v", id, err)
			respond.JSON(w, http.StatusBadGateway, gen.ErrorResponse{
				Error: "could not remove the container from the server, so the service was kept: " + err.Error(),
			})
			return
		}
	}

	if err := db.DeleteService(r.Context(), id); err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to delete service"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// hostPortOrZero flattens "not published" to 0 for the deploy job, which has no
// use for the distinction beyond publish-or-not.
func hostPortOrZero(v *int32) int {
	if v == nil {
		return 0
	}
	return int(*v)
}

func intPtrFromInt32Ptr(v *int32) *int {
	if v == nil {
		return nil
	}
	i := int(*v)
	return &i
}

// serviceResponses maps a whole list and decorates each entry with its pending
// change count in one batched pass — three queries for the list, not three per
// service. The canvas asks for every service in an environment at once, so the
// per-service form in a loop is exactly what this exists to avoid.
func serviceResponses(ctx context.Context, svcs []db.Service) ([]gen.ServiceResponse, error) {
	counts, err := db.PendingChangeCounts(ctx, svcs)
	if err != nil {
		return nil, err
	}
	sources, err := db.ListServiceSources(ctx, serviceIDs(svcs))
	if err != nil {
		return nil, err
	}
	resp := make([]gen.ServiceResponse, len(svcs))
	for i, svc := range svcs {
		resp[i] = serviceToResponse(svc, counts[svc.ID])
		if src, ok := sources[svc.ID]; ok {
			resp[i].Source, err = sourceResponsePtr(src)
			if err != nil {
				return nil, err
			}
		}
	}
	return resp, nil
}

func serviceIDs(svcs []db.Service) []string {
	ids := make([]string, len(svcs))
	for i, svc := range svcs {
		ids[i] = svc.ID
	}
	return ids
}

func sourceResponsePtr(src db.ServiceSource) (*gen.ServiceSourceResponse, error) {
	resp, err := serviceSourceResponse(src)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func serviceResponse(ctx context.Context, svc db.Service) (gen.ServiceResponse, error) {
	resp, err := serviceResponses(ctx, []db.Service{svc})
	if err != nil {
		return gen.ServiceResponse{}, err
	}
	return resp[0], nil
}

func serviceToResponse(svc db.Service, pendingChanges int) gen.ServiceResponse {
	return gen.ServiceResponse{
		Id:            svc.ID,
		Name:          svc.Name,
		Image:         svc.Image,
		ContainerName: svc.ContainerName,
		ContainerPort: int(svc.ContainerPort),
		HostPort:      intPtrFromInt32Ptr(svc.HostPort),
		ServerId:      svc.ServerID,
		Kind:          gen.ServiceResponseKind(svc.Kind),
		ProjectId:     svc.ProjectID,
		EnvironmentId: svc.EnvironmentID,
		CreatedAt:     svc.CreatedAt,
		UpdatedAt:     svc.UpdatedAt,

		PendingChangeCount: pendingChanges,
		HasDeployed:        svc.HasDeployed,
	}
}
