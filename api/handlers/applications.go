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

// validFQDN matches valid hostnames: labels separated by dots, each 1-63 chars, total ≤ 253
var validFQDN = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func uniqueViolationMessage(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && strings.Contains(pgErr.ConstraintName, "fqdn") {
		return "fqdn is already in use by another application"
	}
	return "container_name already in use on this server"
}

func validatePortForMode(port int, fqdn *string) string {
	if port < 1 || port > 65535 {
		return "port must be between 1 and 65535"
	}
	if fqdn == nil && (port == 80 || port == 443) {
		return "ports 80 and 443 are reserved for Uploy proxy; use another port or configure a domain"
	}
	return ""
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

	// Validate FQDN if provided
	var fqdn *string
	if req.Fqdn != nil && *req.Fqdn != "" {
		f := strings.ToLower(strings.TrimSpace(*req.Fqdn))
		if !validFQDN.MatchString(f) || len(f) > 253 {
			respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "fqdn must be a valid domain name (e.g. myapp.example.com)"})
			return
		}
		fqdn = &f
	}
	if msg := validatePortForMode(req.Port, fqdn); msg != "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: msg})
		return
	}

	app, err := db.CreateApplication(r.Context(), req.Name, req.Image, req.ContainerName, int32(req.Port), req.ServerId, sc.WorkspaceID, fqdn)
	if err != nil {
		if isUniqueViolation(err) {
			msg := uniqueViolationMessage(err)
			respond.JSON(w, http.StatusConflict, gen.ErrorResponse{Error: msg})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to create application"})
		}
		return
	}

	respond.JSON(w, http.StatusCreated, applicationToResponse(app))
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
		resp[i] = applicationToResponse(app)
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

	respond.JSON(w, http.StatusOK, applicationToResponse(app))
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

	// FQDN update semantics: nil = keep existing, "" = clear, "domain.com" = set
	fqdn := existing.FQDN // default: keep existing
	if req.Fqdn != nil {
		if *req.Fqdn == "" {
			fqdn = nil // explicitly clear
		} else {
			f := strings.ToLower(strings.TrimSpace(*req.Fqdn))
			if !validFQDN.MatchString(f) || len(f) > 253 {
				respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "fqdn must be a valid domain name (e.g. myapp.example.com)"})
				return
			}
			fqdn = &f
		}
	}
	if msg := validatePortForMode(req.Port, fqdn); msg != "" {
		// Backward compatibility: existing direct-mode apps on port 80/443 may still update
		// other fields as long as they keep the same mode and port.
		if !(existing.FQDN == nil && fqdn == nil && int(existing.Port) == req.Port && (req.Port == 80 || req.Port == 443)) {
			respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: msg})
			return
		}
	}

	app, err := db.UpdateApplication(r.Context(), id, req.Name, req.Image, req.ContainerName, int32(req.Port), req.ServerId, fqdn)
	if err != nil {
		if isUniqueViolation(err) {
			msg := uniqueViolationMessage(err)
			respond.JSON(w, http.StatusConflict, gen.ErrorResponse{Error: msg})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to update application"})
		}
		return
	}

	respond.JSON(w, http.StatusOK, applicationToResponse(app))
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

func applicationToResponse(app db.Application) gen.ApplicationResponse {
	return gen.ApplicationResponse{
		Id:            app.ID,
		Name:          app.Name,
		Image:         app.Image,
		ContainerName: app.ContainerName,
		Port:          int(app.Port),
		ServerId:      app.ServerID,
		Fqdn:          app.FQDN,
		CreatedAt:     app.CreatedAt,
		UpdatedAt:     app.UpdatedAt,
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
