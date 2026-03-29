package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/WahyuS002/uploy/auth"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/respond"
	"github.com/WahyuS002/uploy/ssh"
	gossh "golang.org/x/crypto/ssh"
)

func (s *Server) CreateSSHKey(w http.ResponseWriter, r *http.Request) {
	sc, _ := auth.GetSessionContext(r)

	if sc.WorkspaceRole != "owner" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	var req gen.CreateSSHKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "name is required"})
		return
	}

	if _, err := gossh.ParsePrivateKey([]byte(req.PrivateKey)); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid SSH private key"})
		return
	}

	tx, err := db.Pool.Begin(context.Background())
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to begin transaction"})
		return
	}
	defer tx.Rollback(context.Background())

	key, err := db.CreateSSHKeyTx(context.Background(), tx, req.Name, req.PrivateKey, sc.WorkspaceID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to create SSH key"})
		return
	}

	if err := tx.Commit(context.Background()); err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to commit transaction"})
		return
	}

	pubKey, err := ssh.DerivePublicKey(key.PrivateKey)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to derive public key"})
		return
	}

	respond.JSON(w, http.StatusCreated, gen.SSHKeyResponse{
		Id:        key.ID,
		Name:      key.Name,
		PublicKey: strings.TrimSpace(pubKey),
		CreatedAt: key.CreatedAt,
	})
}

func (s *Server) GenerateSSHKey(w http.ResponseWriter, r *http.Request) {
	sc, _ := auth.GetSessionContext(r)

	if sc.WorkspaceRole != "owner" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	var req gen.GenerateSSHKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "name is required"})
		return
	}

	privateKeyPEM, err := ssh.GenerateEd25519Key()
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to generate keypair"})
		return
	}

	tx, err := db.Pool.Begin(context.Background())
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to begin transaction"})
		return
	}
	defer tx.Rollback(context.Background())

	key, err := db.CreateSSHKeyTx(context.Background(), tx, req.Name, privateKeyPEM, sc.WorkspaceID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to store SSH key"})
		return
	}

	if err := tx.Commit(context.Background()); err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to commit transaction"})
		return
	}

	pubKey, err := ssh.DerivePublicKey(key.PrivateKey)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to derive public key"})
		return
	}

	respond.JSON(w, http.StatusCreated, gen.SSHKeyResponse{
		Id:        key.ID,
		Name:      key.Name,
		PublicKey: strings.TrimSpace(pubKey),
		CreatedAt: key.CreatedAt,
	})
}

func (s *Server) ListSSHKeys(w http.ResponseWriter, r *http.Request) {
	sc, _ := auth.GetSessionContext(r)

	if sc.WorkspaceRole == "owner" {
		keys, err := db.ListSSHKeysByWorkspace(context.Background(), sc.WorkspaceID)
		if err != nil {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list SSH keys"})
			return
		}
		resp := make([]gen.SSHKeyResponse, len(keys))
		for i, k := range keys {
			pubKey, err := ssh.DerivePublicKey(k.PrivateKey)
			if err != nil {
				respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to derive public key"})
				return
			}
			resp[i] = gen.SSHKeyResponse{
				Id:        k.ID,
				Name:      k.Name,
				PublicKey: strings.TrimSpace(pubKey),
				CreatedAt: k.CreatedAt,
			}
		}
		respond.JSON(w, http.StatusOK, resp)
		return
	}

	keys, err := db.ListSSHKeyMetadataByWorkspace(context.Background(), sc.WorkspaceID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list SSH keys"})
		return
	}
	resp := make([]gen.SSHKeyResponse, len(keys))
	for i, k := range keys {
		resp[i] = gen.SSHKeyResponse{
			Id:        k.ID,
			Name:      k.Name,
			CreatedAt: k.CreatedAt,
		}
	}
	respond.JSON(w, http.StatusOK, resp)
}
