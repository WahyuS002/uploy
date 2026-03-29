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

func (s *Server) CreateApplicationDomain(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)

	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

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

	var req gen.CreateDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}

	domain := strings.ToLower(strings.TrimSpace(req.Domain))
	if !validFQDN.MatchString(domain) || len(domain) > 253 {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "domain must be a valid domain name (e.g. myapp.example.com)"})
		return
	}

	// First domain for an app is automatically primary
	existing, err := db.ListDomainsByApplication(r.Context(), id)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to check existing domains"})
		return
	}
	isPrimary := len(existing) == 0

	d, err := db.CreateApplicationDomain(r.Context(), id, domain, isPrimary)
	if err != nil {
		if isUniqueViolation(err) {
			respond.JSON(w, http.StatusConflict, gen.ErrorResponse{Error: "domain is already in use"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to add domain"})
		}
		return
	}

	respond.JSON(w, http.StatusCreated, domainToResponse(d))
}

func (s *Server) ListApplicationDomains(w http.ResponseWriter, r *http.Request, id string) {
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

	domains, err := db.ListDomainsByApplication(r.Context(), id)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list domains"})
		return
	}

	resp := make([]gen.ApplicationDomainResponse, len(domains))
	for i, d := range domains {
		resp[i] = domainToResponse(d)
	}

	respond.JSON(w, http.StatusOK, resp)
}

func (s *Server) UpdateApplicationDomain(w http.ResponseWriter, r *http.Request, id string, domainId string) {
	sc, _ := auth.GetSessionContext(r)

	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

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

	// Verify the domain belongs to this application
	existing, err := db.GetApplicationDomainByID(r.Context(), domainId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "domain not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to get domain"})
		}
		return
	}
	if existing.ApplicationID != id {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "domain not found"})
		return
	}

	var req gen.UpdateDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}

	d, err := db.UpdateApplicationDomainPrimary(r.Context(), id, domainId, req.IsPrimary)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to update domain"})
		return
	}

	respond.JSON(w, http.StatusOK, domainToResponse(d))
}

func (s *Server) DeleteApplicationDomain(w http.ResponseWriter, r *http.Request, id string, domainId string) {
	sc, _ := auth.GetSessionContext(r)

	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

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

	// Verify the domain belongs to this application
	existing, err := db.GetApplicationDomainByID(r.Context(), domainId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "domain not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to get domain"})
		}
		return
	}
	if existing.ApplicationID != id {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "domain not found"})
		return
	}

	if err := db.DeleteApplicationDomain(r.Context(), domainId); err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to delete domain"})
		return
	}

	w.Header().Set("X-Uploy-Action", "redeploy-recommended")
	w.WriteHeader(http.StatusNoContent)
}

func domainToResponse(d db.ApplicationDomain) gen.ApplicationDomainResponse {
	return gen.ApplicationDomainResponse{
		Id:               d.ID,
		Domain:           d.Domain,
		IsPrimary:        d.IsPrimary,
		Status:           gen.ApplicationDomainResponseStatus(d.Status),
		LastError:        d.LastError,
		LastReconciledAt: d.LastReconciledAt,
		ReadyAt:          d.ReadyAt,
		CreatedAt:        d.CreatedAt,
		UpdatedAt:        d.UpdatedAt,
	}
}
