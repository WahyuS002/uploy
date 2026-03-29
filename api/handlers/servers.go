package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/WahyuS002/uploy/auth"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/respond"
	"github.com/WahyuS002/uploy/ssh"
)

func (s *Server) CreateServer(w http.ResponseWriter, r *http.Request) {
	sc, _ := auth.GetSessionContext(r)

	if sc.WorkspaceRole != "owner" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	var req gen.CreateServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "name is required"})
		return
	}
	req.Host = strings.TrimSpace(req.Host)
	if req.Host == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "host is required"})
		return
	}
	req.SshUser = strings.TrimSpace(req.SshUser)
	if req.SshUser == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "ssh_user is required"})
		return
	}
	if req.SshKeyId == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "ssh_key_id is required"})
		return
	}

	port := int32(22)
	if req.Port != nil {
		port = int32(*req.Port)
	}

	ctx := r.Context()

	// Fetch the SSH key to run connectivity test
	key, err := db.GetSSHKeyByID(ctx, req.SshKeyId)
	if err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "SSH key not found"})
		return
	}

	// Verify the key belongs to the same workspace
	if key.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "SSH key does not belong to this workspace"})
		return
	}

	// SSH connectivity test: connect, open session, run a command
	client, err := ssh.NewClient(ssh.ServerConfig{
		Host:       req.Host,
		Port:       int(port),
		User:       req.SshUser,
		PrivateKey: key.PrivateKey,
	})
	if err != nil {
		respond.JSON(w, http.StatusUnprocessableEntity, gen.ErrorResponse{Error: "SSH connection failed: " + err.Error()})
		return
	}
	defer client.Close()

	if err := client.TestSession(); err != nil {
		respond.JSON(w, http.StatusUnprocessableEntity, gen.ErrorResponse{Error: "SSH session test failed: " + err.Error()})
		return
	}

	if err := client.DetectDocker(); err != nil {
		respond.JSON(w, http.StatusUnprocessableEntity, gen.ErrorResponse{Error: err.Error()})
		return
	}

	srv, err := db.CreateServer(ctx, req.Name, req.Host, port, req.SshUser, req.SshKeyId, sc.WorkspaceID)
	if err != nil {
		log.Printf("CreateServer error: %v", err)
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to create server"})
		return
	}

	respond.JSON(w, http.StatusCreated, serverToResponse(srv))
}

func (s *Server) ListServers(w http.ResponseWriter, r *http.Request) {
	sc, _ := auth.GetSessionContext(r)

	servers, err := db.ListServersByWorkspace(r.Context(), sc.WorkspaceID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list servers"})
		return
	}

	resp := make([]gen.ServerResponse, len(servers))
	for i, srv := range servers {
		resp[i] = serverToResponse(srv)
	}

	respond.JSON(w, http.StatusOK, resp)
}

func (s *Server) CheckConnection(w http.ResponseWriter, r *http.Request) {
	sc, _ := auth.GetSessionContext(r)

	if sc.WorkspaceRole != "owner" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	var req gen.CheckConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}

	req.Host = strings.TrimSpace(req.Host)
	if req.Host == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "host is required"})
		return
	}
	req.SshUser = strings.TrimSpace(req.SshUser)
	if req.SshUser == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "ssh_user is required"})
		return
	}
	if req.SshKeyId == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "ssh_key_id is required"})
		return
	}

	port := int32(22)
	if req.Port != nil {
		port = int32(*req.Port)
	}

	ctx := r.Context()

	key, err := db.GetSSHKeyByID(ctx, req.SshKeyId)
	if err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "SSH key not found"})
		return
	}
	if key.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "SSH key does not belong to this workspace"})
		return
	}

	client, err := ssh.NewClient(ssh.ServerConfig{
		Host:       req.Host,
		Port:       int(port),
		User:       req.SshUser,
		PrivateKey: key.PrivateKey,
	})
	if err != nil {
		respond.JSON(w, http.StatusUnprocessableEntity, gen.ErrorResponse{Error: "SSH connection failed: " + err.Error()})
		return
	}
	defer client.Close()

	if err := client.TestSession(); err != nil {
		respond.JSON(w, http.StatusUnprocessableEntity, gen.ErrorResponse{Error: "SSH session test failed: " + err.Error()})
		return
	}

	if err := client.DetectDocker(); err != nil {
		respond.JSON(w, http.StatusUnprocessableEntity, gen.ErrorResponse{Error: err.Error()})
		return
	}

	respond.JSON(w, http.StatusOK, gen.CheckConnectionResponse{Ok: true})
}

func serverToResponse(srv db.AppServer) gen.ServerResponse {
	return gen.ServerResponse{
		Id:                    srv.ID,
		Name:                  srv.Name,
		Host:                  srv.Host,
		Port:                  int(srv.Port),
		SshUser:               srv.SSHUser,
		SshKeyId:              srv.SSHKeyID,
		ProxyStatus:           gen.ServerResponseProxyStatus(srv.ProxyStatus),
		ProxyLastReconciledAt: srv.ProxyLastReconciledAt,
		ProxyLastError:        srv.ProxyLastError,
		CreatedAt:             srv.CreatedAt,
	}
}
