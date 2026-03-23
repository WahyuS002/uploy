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

// requireApp checks that the application exists and belongs to the current workspace.
func (s *Server) requireApp(w http.ResponseWriter, r *http.Request, id string) (db.Application, bool) {
	sc, _ := auth.GetSessionContext(r)

	app, err := db.GetApplicationByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "application not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to get application"})
		}
		return db.Application{}, false
	}
	if app.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "application not found"})
		return db.Application{}, false
	}
	return app, true
}

func (s *Server) ListApplicationEnvs(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)
	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	app, ok := s.requireApp(w, r, id)
	if !ok {
		return
	}

	envs, err := db.ListApplicationEnvs(r.Context(), app.ID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list environment variables"})
		return
	}

	resp := make([]gen.ApplicationEnvResponse, len(envs))
	for i, e := range envs {
		resp[i] = gen.ApplicationEnvResponse{
			Id:        e.ID,
			Key:       e.Key,
			Value:     e.Value,
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
		}
	}

	respond.JSON(w, http.StatusOK, resp)
}

func (s *Server) UpsertApplicationEnv(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)
	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	app, ok := s.requireApp(w, r, id)
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

	env, err := db.UpsertApplicationEnv(r.Context(), app.ID, req.Key, req.Value)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to set environment variable"})
		return
	}

	respond.JSON(w, http.StatusOK, gen.ApplicationEnvResponse{
		Id:        env.ID,
		Key:       env.Key,
		Value:     env.Value,
		CreatedAt: env.CreatedAt,
		UpdatedAt: env.UpdatedAt,
	})
}

func (s *Server) DeleteApplicationEnv(w http.ResponseWriter, r *http.Request, id string, key string) {
	sc, _ := auth.GetSessionContext(r)
	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	_, ok := s.requireApp(w, r, id)
	if !ok {
		return
	}

	if err := db.DeleteApplicationEnv(r.Context(), id, key); err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to delete environment variable"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
