package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/WahyuS002/uploy/auth"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/respond"
	"github.com/WahyuS002/uploy/ssh"
)

// probeFailure is an SSH probe failure tagged with the stage that failed.
// The stages need entirely different fixes — a wrong host is a network
// problem, a refused key is an authorized_keys problem, a missing docker is a
// provisioning problem — so the client gets a code to branch on rather than
// having to pattern-match English.
type probeFailure struct {
	code    gen.ErrorResponseCode
	message string
}

// probeServer opens an SSH connection and verifies the host can run commands
// and reach docker. On success the caller owns closing the client.
func probeServer(host string, port int, user, privateKey string) (*ssh.Client, *probeFailure) {
	client, err := ssh.NewClient(ssh.ServerConfig{
		Host:       host,
		Port:       port,
		User:       user,
		PrivateKey: privateKey,
	})
	if err != nil {
		return nil, classifyDialError(err)
	}

	if err := client.TestSession(); err != nil {
		client.Close()
		return nil, &probeFailure{
			code:    gen.SessionFailed,
			message: "Connected, but the server could not run a command: " + err.Error(),
		}
	}

	if err := client.DetectDocker(); err != nil {
		client.Close()
		return nil, &probeFailure{code: gen.DockerMissing, message: err.Error()}
	}

	return client, nil
}

// classifyDialError separates the two failures that NewClient collapses into
// one return. gossh.Dial wraps a *net.OpError only when TCP itself never
// landed; once the socket is open, any remaining failure is the handshake,
// which in practice means the server would not accept our key.
func classifyDialError(err error) *probeFailure {
	if errors.Is(err, ssh.ErrInvalidPrivateKey) {
		return &probeFailure{
			code:    gen.KeyInvalid,
			message: "The stored SSH key could not be parsed: " + err.Error(),
		}
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return &probeFailure{
			code:    gen.Unreachable,
			message: "Could not reach the server: " + err.Error(),
		}
	}

	return &probeFailure{
		code:    gen.KeyRejected,
		message: "SSH authentication failed: " + err.Error(),
	}
}

// resolveWorkspaceKey loads an SSH key and confirms it belongs to the caller's
// workspace. The bool reports whether a response has already been written.
func (s *Server) resolveWorkspaceKey(w http.ResponseWriter, r *http.Request, keyID, workspaceID string) (db.SSHKey, bool) {
	key, err := db.GetSSHKeyByID(r.Context(), keyID)
	if err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "SSH key not found"})
		return db.SSHKey{}, false
	}
	if key.WorkspaceID != workspaceID {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "SSH key does not belong to this workspace"})
		return db.SSHKey{}, false
	}
	return key, true
}

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

	key, ok := s.resolveWorkspaceKey(w, r, req.SshKeyId, sc.WorkspaceID)
	if !ok {
		return
	}

	// The probe is the gate: an unreachable server never becomes a record, so
	// the client does not need to pre-verify before submitting.
	client, probeErr := probeServer(req.Host, int(port), req.SshUser, key.PrivateKey)
	if probeErr != nil {
		code := probeErr.code
		respond.JSON(w, http.StatusUnprocessableEntity, gen.ErrorResponse{
			Error: probeErr.message,
			Code:  &code,
		})
		return
	}
	defer client.Close()

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

	key, ok := s.resolveWorkspaceKey(w, r, req.SshKeyId, sc.WorkspaceID)
	if !ok {
		return
	}

	client, probeErr := probeServer(req.Host, int(port), req.SshUser, key.PrivateKey)
	if probeErr != nil {
		code := probeErr.code
		respond.JSON(w, http.StatusUnprocessableEntity, gen.ErrorResponse{
			Error: probeErr.message,
			Code:  &code,
		})
		return
	}
	defer client.Close()

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
