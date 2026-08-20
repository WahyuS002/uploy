package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/WahyuS002/uploy/auth"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/respond"
	"github.com/WahyuS002/uploy/source"
	"github.com/jackc/pgx/v5"
)

func (s *Server) CreateServiceFromSource(w http.ResponseWriter, r *http.Request) {
	sc, _ := auth.GetSessionContext(r)
	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	var req gen.CreateServiceFromSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "name is required"})
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
	if msg := validatePorts(req.ContainerPort, req.HostPort); msg != "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: msg})
		return
	}

	repo, err := source.ParseRepoURL(req.RepoUrl)
	if err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "repo_url must be a public github.com repository"})
		return
	}
	repo.Branch = "main"
	if req.Branch != nil && strings.TrimSpace(*req.Branch) != "" {
		repo.Branch = strings.TrimSpace(*req.Branch)
	}

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

	env, err := db.GetEnvironmentByID(r.Context(), req.EnvironmentId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "environment not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to look up environment"})
		}
		return
	}
	project, err := db.GetProjectByID(r.Context(), env.ProjectID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to look up project"})
		return
	}
	if project.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "environment not found"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), sourceAnalysisTimeout)
	defer cancel()
	analysis, err := s.analyzeSource(ctx, repo)
	if err != nil {
		s.respondSourceAnalysisError(w, ctx, repo, err)
		return
	}
	detected, err := json.Marshal(analysis.Info)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to store source detection"})
		return
	}

	svc, err := db.CreateServiceFromSource(
		r.Context(), req.Name, req.ContainerName, int32(req.ContainerPort), int32PtrFromIntPtr(req.HostPort),
		req.ServerId, sc.WorkspaceID, project.ID, req.EnvironmentId,
		"github", repo.Owner, repo.Name, repo.Branch, detected,
	)
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
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to read service state"})
		return
	}
	respond.JSON(w, http.StatusCreated, resp)
}

func serviceSourceResponse(src db.ServiceSource) (gen.ServiceSourceResponse, error) {
	var info source.Info
	if len(src.Detected) > 0 {
		if err := json.Unmarshal(src.Detected, &info); err != nil {
			return gen.ServiceSourceResponse{}, err
		}
	}
	runtimes := info.RuntimeVersions
	if runtimes == nil {
		runtimes = map[string]string{}
	}
	detection := gen.SourceDetection{Provider: info.Provider, RuntimeVersions: runtimes}
	if info.StartCommand != "" {
		startCommand := info.StartCommand
		detection.StartCommand = &startCommand
	}
	return gen.ServiceSourceResponse{
		Owner:    src.Owner,
		Repo:     src.Repo,
		Branch:   src.Branch,
		Detected: detection,
	}, nil
}
