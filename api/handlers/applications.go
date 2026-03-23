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

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (s *Server) CreateApplication(w http.ResponseWriter, r *http.Request) {
	sc, _ := auth.GetSessionContext(r)

	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	var req gen.CreateApplicationRequest
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

	// Verifikasi server ada dan milik workspace yang sama
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

	app, err := db.CreateApplication(r.Context(), req.Name, req.Image, req.ContainerName, int32(req.Port), req.ServerId, sc.WorkspaceID)
	if err != nil {
		if isUniqueViolation(err) {
			respond.JSON(w, http.StatusConflict, gen.ErrorResponse{Error: "container_name already in use on this server"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to create application"})
		}
		return
	}

	respond.JSON(w, http.StatusCreated, gen.ApplicationResponse{
		Id:            app.ID,
		Name:          app.Name,
		Image:         app.Image,
		ContainerName: app.ContainerName,
		Port:          int(app.Port),
		ServerId:      app.ServerID,
		CreatedAt:     app.CreatedAt,
		UpdatedAt:     app.UpdatedAt,
	})
}

func (s *Server) ListApplications(w http.ResponseWriter, r *http.Request) {
	sc, _ := auth.GetSessionContext(r)

	apps, err := db.ListApplicationsByWorkspace(r.Context(), sc.WorkspaceID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list applications"})
		return
	}

	resp := make([]gen.ApplicationResponse, len(apps))
	for i, app := range apps {
		resp[i] = gen.ApplicationResponse{
			Id:            app.ID,
			Name:          app.Name,
			Image:         app.Image,
			ContainerName: app.ContainerName,
			Port:          int(app.Port),
			ServerId:      app.ServerID,
			CreatedAt:     app.CreatedAt,
			UpdatedAt:     app.UpdatedAt,
		}
	}

	respond.JSON(w, http.StatusOK, resp)
}

func (s *Server) GetApplication(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)

	app, err := db.GetApplicationByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "application not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to get application"})
		}
		return
	}

	if app.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "application not found"})
		return
	}

	respond.JSON(w, http.StatusOK, gen.ApplicationResponse{
		Id:            app.ID,
		Name:          app.Name,
		Image:         app.Image,
		ContainerName: app.ContainerName,
		Port:          int(app.Port),
		ServerId:      app.ServerID,
		CreatedAt:     app.CreatedAt,
		UpdatedAt:     app.UpdatedAt,
	})
}

func (s *Server) UpdateApplication(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)

	// Cek app ada dan milik workspace
	existing, err := db.GetApplicationByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "application not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to get application"})
		}
		return
	}
	if existing.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "application not found"})
		return
	}

	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	var req gen.UpdateApplicationRequest
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

	// Mengganti container_name atau server_id akan meninggalkan container lama tetap hidup
	// karena deploy hanya tahu konfigurasi terbaru. Tolak perubahan ini.
	if req.ContainerName != existing.ContainerName {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "container_name cannot be changed; delete and recreate the application instead"})
		return
	}
	if req.ServerId != existing.ServerID {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "server_id cannot be changed; delete and recreate the application instead"})
		return
	}

	app, err := db.UpdateApplication(r.Context(), id, req.Name, req.Image, req.ContainerName, int32(req.Port), req.ServerId)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to update application"})
		return
	}

	respond.JSON(w, http.StatusOK, gen.ApplicationResponse{
		Id:            app.ID,
		Name:          app.Name,
		Image:         app.Image,
		ContainerName: app.ContainerName,
		Port:          int(app.Port),
		ServerId:      app.ServerID,
		CreatedAt:     app.CreatedAt,
		UpdatedAt:     app.UpdatedAt,
	})
}

func (s *Server) DeleteApplication(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)

	app, err := db.GetApplicationByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "application not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to get application"})
		}
		return
	}
	if app.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "application not found"})
		return
	}

	if sc.WorkspaceRole != "owner" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	if err := db.DeleteApplication(r.Context(), id); err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to delete application"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
