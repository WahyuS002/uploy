package db

import (
	"context"
	"fmt"
	"time"

	"github.com/WahyuS002/uploy/crypto"
	"github.com/WahyuS002/uploy/db/sqlcgen"
	"github.com/jackc/pgx/v5"
)

type SSHKey struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	PrivateKey  string    `json:"-"`
	WorkspaceID string    `json:"workspace_id"`
	CreatedAt   time.Time `json:"created_at"`
}

func sshKeyFromGen(k sqlcgen.SshKey) SSHKey {
	return SSHKey{
		ID:          k.ID,
		Name:        k.Name,
		PrivateKey:  k.PrivateKey,
		WorkspaceID: k.WorkspaceID,
		CreatedAt:   k.CreatedAt,
	}
}

func CreateSSHKeyTx(ctx context.Context, tx pgx.Tx, name, privateKey, workspaceID string) (SSHKey, error) {
	encrypted, err := crypto.Encrypt(privateKey)
	if err != nil {
		return SSHKey{}, fmt.Errorf("encrypt private key: %w", err)
	}
	k, err := sqlcgen.New(tx).CreateSSHKey(ctx, sqlcgen.CreateSSHKeyParams{
		Name:        name,
		PrivateKey:  encrypted,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return SSHKey{}, err
	}
	// Return the original plaintext, not the encrypted value from DB
	result := sshKeyFromGen(k)
	result.PrivateKey = privateKey
	return result, nil
}

func GetSSHKeyByID(ctx context.Context, id string) (SSHKey, error) {
	k, err := Queries.GetSSHKeyByID(ctx, id)
	if err != nil {
		return SSHKey{}, err
	}
	result := sshKeyFromGen(k)
	result.PrivateKey, err = crypto.Decrypt(result.PrivateKey)
	if err != nil {
		return SSHKey{}, fmt.Errorf("decrypt private key: %w", err)
	}
	return result, nil
}

// ListSSHKeysByWorkspace fetches keys with decrypted private keys (for owner flows that derive public keys).
func ListSSHKeysByWorkspace(ctx context.Context, workspaceID string) ([]SSHKey, error) {
	rows, err := Queries.ListSSHKeysByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	keys := make([]SSHKey, len(rows))
	for i, r := range rows {
		decrypted, err := crypto.Decrypt(r.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("decrypt private key for %s: %w", r.ID, err)
		}
		keys[i] = SSHKey{
			ID:          r.ID,
			Name:        r.Name,
			PrivateKey:  decrypted,
			WorkspaceID: r.WorkspaceID,
			CreatedAt:   r.CreatedAt,
		}
	}
	return keys, nil
}

// ListSSHKeyMetadataByWorkspace fetches keys without private key material (for non-owner listing).
func ListSSHKeyMetadataByWorkspace(ctx context.Context, workspaceID string) ([]SSHKey, error) {
	rows, err := Queries.ListSSHKeyMetadataByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	keys := make([]SSHKey, len(rows))
	for i, r := range rows {
		keys[i] = SSHKey{
			ID:          r.ID,
			Name:        r.Name,
			WorkspaceID: r.WorkspaceID,
			CreatedAt:   r.CreatedAt,
		}
	}
	return keys, nil
}
