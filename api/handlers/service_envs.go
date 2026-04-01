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
)

// validEnvKey matches POSIX env var names: [A-Za-z_][A-Za-z0-9_]*
var validEnvKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// requireService checks that the service exists and belongs to the current workspace.
func (s *Server) requireService(w http.ResponseWriter, r *http.Request, id string) (db.Service, bool) {
	sc, _ := auth.GetSessionContext(r)

	svc, err := db.GetServiceByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "service not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to get service"})
		}
		return db.Service{}, false
	}
	if svc.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "service not found"})
		return db.Service{}, false
	}
	return svc, true
}

func (s *Server) ListServiceEnvs(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)
	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	svc, ok := s.requireService(w, r, id)
	if !ok {
		return
	}

	envs, err := db.ListServiceEnvVars(r.Context(), svc.ID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list environment variables"})
		return
	}

	resp := make([]gen.ServiceEnvResponse, len(envs))
	for i, e := range envs {
		resp[i] = gen.ServiceEnvResponse{
			Id:        e.ID,
			Key:       e.Key,
			Value:     e.Value,
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
		}
	}

	respond.JSON(w, http.StatusOK, resp)
}

func (s *Server) UpsertServiceEnv(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)
	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	svc, ok := s.requireService(w, r, id)
	if !ok {
		return
	}

	var req gen.UpsertEnvRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}

	req.Key = strings.TrimSpace(req.Key)
	if req.Key == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "key is required"})
		return
	}
	if !validEnvKey.MatchString(req.Key) {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "key must match [A-Za-z_][A-Za-z0-9_]*"})
		return
	}

	env, err := db.UpsertServiceEnvVar(r.Context(), svc.ID, req.Key, req.Value)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to set environment variable"})
		return
	}

	respond.JSON(w, http.StatusOK, gen.ServiceEnvResponse{
		Id:        env.ID,
		Key:       env.Key,
		Value:     env.Value,
		CreatedAt: env.CreatedAt,
		UpdatedAt: env.UpdatedAt,
	})
}

func (s *Server) DeleteServiceEnv(w http.ResponseWriter, r *http.Request, id string, key string) {
	sc, _ := auth.GetSessionContext(r)
	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	_, ok := s.requireService(w, r, id)
	if !ok {
		return
	}

	if err := db.DeleteServiceEnvVar(r.Context(), id, key); err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to delete environment variable"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
