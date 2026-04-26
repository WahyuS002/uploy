package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/WahyuS002/uploy/auth"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/respond"
	"github.com/jackc/pgx/v5"
)

func (s *Server) CreateProject(w http.ResponseWriter, r *http.Request) {
	sc, _ := auth.GetSessionContext(r)

	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	var req gen.CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}

	name := ""
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}

	proj, _, err := db.CreateProjectWithDefaultEnvironment(r.Context(), name, sc.WorkspaceID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to create project"})
		return
	}

	respond.JSON(w, http.StatusCreated, projectToResponse(proj))
}

func (s *Server) CreateProjectFromImage(w http.ResponseWriter, r *http.Request) {
	sc, _ := auth.GetSessionContext(r)

	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	var req gen.CreateProjectFromImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}

	image := strings.TrimSpace(req.Image)
	if image == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "image is required"})
		return
	}
	if !validImage.MatchString(image) {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "image contains invalid characters"})
		return
	}

	if msg := validatePort(req.Port); msg != "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: msg})
		return
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

	svcName := serviceNameFromImage(image)
	containerName := containerNameFromImage(image)
	if !validContainerName.MatchString(containerName) {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "could not derive a valid container name from the image"})
		return
	}

	proj, env, svc, err := db.CreateProjectWithDefaultEnvironmentAndService(
		r.Context(),
		sc.WorkspaceID,
		svcName,
		image,
		containerName,
		int32(req.Port),
		req.ServerId,
		"application",
	)
	if err != nil {
		if isUniqueViolation(err) {
			respond.JSON(w, http.StatusConflict, gen.ErrorResponse{Error: "container_name already in use on this server"})
			return
		}
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to create project"})
		return
	}

	respond.JSON(w, http.StatusCreated, gen.CreateProjectFromImageResponse{
		Project:     projectToResponse(proj),
		Environment: environmentToResponse(env),
		Service:     serviceToResponse(svc),
	})
}

func (s *Server) ListProjects(w http.ResponseWriter, r *http.Request) {
	sc, _ := auth.GetSessionContext(r)

	projects, err := db.ListProjectsByWorkspace(r.Context(), sc.WorkspaceID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list projects"})
		return
	}

	resp := make([]gen.ProjectResponse, len(projects))
	for i, p := range projects {
		resp[i] = projectToResponse(p)
	}

	respond.JSON(w, http.StatusOK, resp)
}

func (s *Server) GetProject(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)

	proj, err := db.GetProjectByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "project not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to get project"})
		}
		return
	}
	if proj.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "project not found"})
		return
	}

	respond.JSON(w, http.StatusOK, projectToResponse(proj))
}

func (s *Server) UpdateProject(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)

	proj, err := db.GetProjectByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "project not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to get project"})
		}
		return
	}
	if proj.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "project not found"})
		return
	}

	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	var req gen.UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "name is required"})
		return
	}

	updated, err := db.UpdateProject(r.Context(), id, req.Name)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to update project"})
		return
	}

	respond.JSON(w, http.StatusOK, projectToResponse(updated))
}

func (s *Server) DeleteProject(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)

	proj, err := db.GetProjectByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "project not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to get project"})
		}
		return
	}
	if proj.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "project not found"})
		return
	}

	if sc.WorkspaceRole != "owner" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	if err := db.DeleteProject(r.Context(), id); err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to delete project"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// serviceNameFromImage derives a human-readable service name from a Docker
// image reference. It strips registry, repository path, tag, and digest and
// falls back to "service" when no usable segment remains.
func serviceNameFromImage(image string) string {
	ref := image
	if i := strings.Index(ref, "@"); i >= 0 {
		ref = ref[:i]
	}
	if i := strings.LastIndex(ref, ":"); i >= 0 {
		// only treat as tag if the colon is in the last path segment
		if !strings.ContainsAny(ref[i:], "/") {
			ref = ref[:i]
		}
	}
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		ref = ref[i+1:]
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "service"
	}
	return ref
}

// containerNameFromImage builds a Docker-safe container name with a short
// random suffix to avoid collisions on the same server when two services
// derived from the same image are deployed.
func containerNameFromImage(image string) string {
	base := serviceNameFromImage(image)
	var b strings.Builder
	for i, r := range base {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		case r == '_' || r == '.' || r == '-':
			if i == 0 {
				continue
			}
			b.WriteRune(r)
		}
	}
	cleaned := strings.Trim(b.String(), "._-")
	if cleaned == "" {
		cleaned = "service"
	}

	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		// extremely unlikely; fall back to a static marker that still
		// satisfies validContainerName
		return cleaned + "-x"
	}
	return cleaned + "-" + hex.EncodeToString(suffix)
}

func projectToResponse(p db.Project) gen.ProjectResponse {
	return gen.ProjectResponse{
		Id:          p.ID,
		Name:        p.Name,
		WorkspaceId: p.WorkspaceID,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
