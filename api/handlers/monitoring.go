package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/WahyuS002/uploy/auth"
	"github.com/WahyuS002/uploy/config"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/monitoring"
	"github.com/WahyuS002/uploy/respond"
	"github.com/WahyuS002/uploy/ssh"
	"github.com/jackc/pgx/v5"
)

func (s *Server) ConfigureServerMonitoring(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)
	if sc.WorkspaceRole != "owner" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}
	server, ok := s.workspaceServerWithKey(w, r, id, sc.WorkspaceID)
	if !ok {
		return
	}

	var req gen.ConfigureServerMonitoringRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}
	port := monitoring.AgentPort
	if server.Monitoring.Port != 0 {
		port = int(server.Monitoring.Port)
	}
	if req.Port != nil {
		port = *req.Port
	}
	retentionDays := 7
	if server.Monitoring.RetentionDays != 0 {
		retentionDays = int(server.Monitoring.RetentionDays)
	}
	if req.RetentionDays != nil {
		retentionDays = *req.RetentionDays
	}
	fqdn := monitoringFQDN(server)
	if req.Fqdn != nil {
		fqdn = strings.ToLower(strings.TrimSpace(*req.Fqdn))
	}
	if fqdn != "" && (!validFQDN.MatchString(fqdn) || len(fqdn) > 253) {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "fqdn must be a valid domain name"})
		return
	}
	readerToken := server.ReaderToken
	if req.ReaderToken != nil {
		readerToken = strings.TrimSpace(*req.ReaderToken)
	}
	if len(readerToken) < 32 || len(readerToken) > 512 {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "reader_token must contain 32 to 512 characters"})
		return
	}
	controlToken := server.ControlToken
	if controlToken == "" {
		var err error
		controlToken, err = generateMonitoringToken()
		if err != nil {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to generate monitoring token"})
			return
		}
	}

	agentConfig := monitoring.Config{
		Image:         config.C.MonitoringImage,
		HostPort:      port,
		RetentionDays: retentionDays,
		FQDN:          fqdn,
		ControlToken:  controlToken,
		ReaderToken:   readerToken,
	}
	if err := monitoring.ValidateConfig(agentConfig); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: err.Error()})
		return
	}
	if fqdn != "" {
		used, err := db.MonitoringFQDNInUse(r.Context(), id, fqdn)
		if err != nil {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to validate monitoring fqdn"})
			return
		}
		if used {
			respond.JSON(w, http.StatusConflict, gen.ErrorResponse{Error: "fqdn is already in use"})
			return
		}
	}

	desired := db.MonitoringConfig{
		Enabled: true, Port: int32(port), RetentionDays: int32(retentionDays),
		FQDN: fqdn, Status: "provisioning",
	}
	previous := monitoringDBConfig(server)
	if err := db.SetServerMonitoring(r.Context(), id, desired, controlToken, readerToken); err != nil {
		if isUniqueViolation(err) {
			respond.JSON(w, http.StatusConflict, gen.ErrorResponse{Error: "fqdn is already in use"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to save monitoring configuration"})
		}
		return
	}

	client, err := monitoringClient(server)
	if err == nil {
		defer client.Close()
		err = monitoring.Enable(r.Context(), client, id, agentConfig)
	}
	if err != nil {
		previous.Status = "error"
		previous.LastError = err.Error()
		_ = db.SetServerMonitoring(r.Context(), id, previous, server.ControlToken, server.ReaderToken)
		respond.JSON(w, http.StatusUnprocessableEntity, gen.ErrorResponse{Error: "monitoring setup failed: " + err.Error()})
		return
	}
	desired.Status = "ready"
	if err := db.SetServerMonitoring(r.Context(), id, desired, controlToken, readerToken); err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "monitoring started but status update failed"})
		return
	}
	updated, err := db.GetServerByID(r.Context(), id)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "monitoring started but server reload failed"})
		return
	}
	respond.JSON(w, http.StatusOK, serverToResponse(updated))
}

func (s *Server) DisableServerMonitoring(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)
	if sc.WorkspaceRole != "owner" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}
	server, ok := s.workspaceServerWithKey(w, r, id, sc.WorkspaceID)
	if !ok {
		return
	}
	if !server.Monitoring.Enabled {
		current, err := db.GetServerByID(r.Context(), id)
		if err != nil {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to reload server"})
			return
		}
		respond.JSON(w, http.StatusOK, serverToResponse(current))
		return
	}
	client, err := monitoringClient(server)
	if err == nil {
		defer client.Close()
		err = monitoring.Disable(r.Context(), client, id)
	}
	state := monitoringDBConfig(server)
	if err != nil {
		state.Status = "error"
		state.LastError = err.Error()
		_ = db.SetServerMonitoring(r.Context(), id, state, server.ControlToken, server.ReaderToken)
		respond.JSON(w, http.StatusUnprocessableEntity, gen.ErrorResponse{Error: "monitoring disable failed: " + err.Error()})
		return
	}
	cleanupAt := time.Now().UTC().Add(7 * 24 * time.Hour)
	state.Enabled = false
	state.Status = "disabled"
	state.CleanupAt = &cleanupAt
	if err := db.SetServerMonitoring(r.Context(), id, state, server.ControlToken, server.ReaderToken); err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "monitoring stopped but status update failed"})
		return
	}
	updated, err := db.GetServerByID(r.Context(), id)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "monitoring stopped but server reload failed"})
		return
	}
	respond.JSON(w, http.StatusOK, serverToResponse(updated))
}

func (s *Server) DeleteServerMonitoringHistory(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)
	if sc.WorkspaceRole != "owner" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}
	server, ok := s.workspaceServerWithKey(w, r, id, sc.WorkspaceID)
	if !ok {
		return
	}
	var err error
	if server.Monitoring.Enabled {
		err = monitoring.DeleteHistory(r.Context(), monitoringTarget(server), server.ControlToken)
	} else {
		var client *ssh.Client
		client, err = monitoringClient(server)
		if err == nil {
			defer client.Close()
			err = monitoring.DeleteLocalData(r.Context(), client)
		}
	}
	if err != nil {
		respond.JSON(w, http.StatusBadGateway, gen.ErrorResponse{Error: "failed to delete monitoring history: " + err.Error()})
		return
	}
	if !server.Monitoring.Enabled {
		if err := db.ClearServerMonitoringData(r.Context(), id); err != nil {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "history deleted but cleanup state update failed"})
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) workspaceServerWithKey(w http.ResponseWriter, r *http.Request, id, workspaceID string) (db.ServerWithKey, bool) {
	server, err := db.GetServerWithKey(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "server not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to look up server"})
		}
		return db.ServerWithKey{}, false
	}
	if server.WorkspaceID != workspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "server not found"})
		return db.ServerWithKey{}, false
	}
	return server, true
}

func monitoringClient(server db.ServerWithKey) (*ssh.Client, error) {
	client, err := ssh.NewClient(ssh.ServerConfig{Host: server.Host, Port: int(server.Port), User: server.SSHUser, PrivateKey: server.PrivateKey})
	if err != nil {
		return nil, err
	}
	if err := client.DetectDocker(); err != nil {
		client.Close()
		return nil, err
	}
	return client, nil
}

// monitoringTarget describes how to reach a server's agent: through the SSH
// connection Uploy already uses for that server, to the loopback port the agent
// publishes on.
func monitoringTarget(server db.ServerWithKey) monitoring.Target {
	return monitoring.Target{
		SSH:  ssh.ServerConfig{Host: server.Host, Port: int(server.Port), User: server.SSHUser, PrivateKey: server.PrivateKey},
		Port: int(server.Monitoring.Port),
	}
}

func monitoringFQDN(server db.ServerWithKey) string {
	if server.Monitoring.FQDN == nil {
		return ""
	}
	return *server.Monitoring.FQDN
}

func monitoringDBConfig(server db.ServerWithKey) db.MonitoringConfig {
	return db.MonitoringConfig{
		Enabled: server.Monitoring.Enabled, Port: server.Monitoring.Port, RetentionDays: server.Monitoring.RetentionDays,
		FQDN: monitoringFQDN(server), Status: server.Monitoring.Status,
		CleanupAt: server.Monitoring.CleanupAt,
	}
}

func generateMonitoringToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
