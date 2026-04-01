package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/WahyuS002/uploy/auth"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/respond"
	"github.com/jackc/pgx/v5"
)

// requireProject checks that the project exists and belongs to the current workspace.
func (s *Server) requireProject(w http.ResponseWriter, r *http.Request, id string) (db.Project, bool) {
	sc, _ := auth.GetSessionContext(r)

	proj, err := db.GetProjectByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "project not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to get project"})
		}
		return db.Project{}, false
	}
	if proj.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "project not found"})
		return db.Project{}, false
	}
	return proj, true
}

func (s *Server) CreateEnvironment(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)

	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	proj, ok := s.requireProject(w, r, id)
	if !ok {
		return
	}

	var req gen.CreateEnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "name is required"})
		return
	}

	env, err := db.CreateEnvironment(r.Context(), req.Name, proj.ID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to create environment"})
		return
	}

	respond.JSON(w, http.StatusCreated, environmentToResponse(env))
}

func (s *Server) ListEnvironments(w http.ResponseWriter, r *http.Request, id string) {
	proj, ok := s.requireProject(w, r, id)
	if !ok {
		return
	}

	envs, err := db.ListEnvironmentsByProject(r.Context(), proj.ID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list environments"})
		return
	}

	resp := make([]gen.EnvironmentResponse, len(envs))
	for i, e := range envs {
		resp[i] = environmentToResponse(e)
	}

	respond.JSON(w, http.StatusOK, resp)
}

func (s *Server) GetEnvironment(w http.ResponseWriter, r *http.Request, id string, envId string) {
	_, ok := s.requireProject(w, r, id)
	if !ok {
		return
	}

	env, err := db.GetEnvironmentByID(r.Context(), envId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "environment not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to get environment"})
		}
		return
	}
	if env.ProjectID != id {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "environment not found"})
		return
	}

	respond.JSON(w, http.StatusOK, environmentToResponse(env))
}

func (s *Server) UpdateEnvironment(w http.ResponseWriter, r *http.Request, id string, envId string) {
	sc, _ := auth.GetSessionContext(r)

	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	_, ok := s.requireProject(w, r, id)
	if !ok {
		return
	}

	env, err := db.GetEnvironmentByID(r.Context(), envId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "environment not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to get environment"})
		}
		return
	}
	if env.ProjectID != id {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "environment not found"})
		return
	}

	var req gen.UpdateEnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "name is required"})
		return
	}

	updated, err := db.UpdateEnvironment(r.Context(), envId, req.Name)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to update environment"})
		return
	}

	respond.JSON(w, http.StatusOK, environmentToResponse(updated))
}

func (s *Server) DeleteEnvironment(w http.ResponseWriter, r *http.Request, id string, envId string) {
	sc, _ := auth.GetSessionContext(r)

	if sc.WorkspaceRole != "owner" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	_, ok := s.requireProject(w, r, id)
	if !ok {
		return
	}

	env, err := db.GetEnvironmentByID(r.Context(), envId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "environment not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to get environment"})
		}
		return
	}
	if env.ProjectID != id {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "environment not found"})
		return
	}

	if err := db.DeleteEnvironment(r.Context(), envId); err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to delete environment"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func environmentToResponse(e db.Environment) gen.EnvironmentResponse {
	return gen.EnvironmentResponse{
		Id:        e.ID,
		Name:      e.Name,
		ProjectId: e.ProjectID,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}
