package db

import (
	"context"
	"fmt"
	"time"

	"github.com/WahyuS002/uploy/crypto"
	"github.com/WahyuS002/uploy/db/sqlcgen"
	"github.com/jackc/pgx/v5/pgtype"
)

type AppServer struct {
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	Host               string     `json:"host"`
	Port               int32      `json:"port"`
	SSHUser            string     `json:"ssh_user"`
	SSHKeyID           string     `json:"ssh_key_id"`
	WorkspaceID        string     `json:"workspace_id"`
	ProxyStatus        string     `json:"proxy_status"`
	ProxyMode          string     `json:"proxy_mode"`
	ProxyLastCheckedAt *time.Time `json:"proxy_last_checked_at"`
	ProxyLastError     *string    `json:"proxy_last_error"`
	CreatedAt          time.Time  `json:"created_at"`
}

type ServerWithKey struct {
	AppServer
	PrivateKey string `json:"-"`
}

func timeToPtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func CreateServer(ctx context.Context, name, host string, port int32, sshUser, sshKeyID, workspaceID string) (AppServer, error) {
	s, err := Queries.CreateServer(ctx, sqlcgen.CreateServerParams{
		Name:        name,
		Host:        host,
		Port:        port,
		SshUser:     sshUser,
		SshKeyID:    sshKeyID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return AppServer{}, err
	}
	return AppServer{
		ID:                 s.ID,
		Name:               s.Name,
		Host:               s.Host,
		Port:               s.Port,
		SSHUser:            s.SshUser,
		SSHKeyID:           s.SshKeyID,
		WorkspaceID:        s.WorkspaceID,
		ProxyStatus:        s.ProxyStatus,
		ProxyMode:          s.ProxyMode,
		ProxyLastCheckedAt: timeToPtr(s.ProxyLastCheckedAt),
		ProxyLastError:     pgTextToStringPtr(s.ProxyLastError),
		CreatedAt:          s.CreatedAt,
	}, nil
}

func GetServerByID(ctx context.Context, id string) (AppServer, error) {
	s, err := Queries.GetServerByID(ctx, id)
	if err != nil {
		return AppServer{}, err
	}
	return AppServer{
		ID:                 s.ID,
		Name:               s.Name,
		Host:               s.Host,
		Port:               s.Port,
		SSHUser:            s.SshUser,
		SSHKeyID:           s.SshKeyID,
		WorkspaceID:        s.WorkspaceID,
		ProxyStatus:        s.ProxyStatus,
		ProxyMode:          s.ProxyMode,
		ProxyLastCheckedAt: timeToPtr(s.ProxyLastCheckedAt),
		ProxyLastError:     pgTextToStringPtr(s.ProxyLastError),
		CreatedAt:          s.CreatedAt,
	}, nil
}

func ListServersByWorkspace(ctx context.Context, workspaceID string) ([]AppServer, error) {
	rows, err := Queries.ListServersByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	servers := make([]AppServer, len(rows))
	for i, r := range rows {
		servers[i] = AppServer{
			ID:                 r.ID,
			Name:               r.Name,
			Host:               r.Host,
			Port:               r.Port,
			SSHUser:            r.SshUser,
			SSHKeyID:           r.SshKeyID,
			WorkspaceID:        r.WorkspaceID,
			ProxyStatus:        r.ProxyStatus,
			ProxyMode:          r.ProxyMode,
			ProxyLastCheckedAt: timeToPtr(r.ProxyLastCheckedAt),
			ProxyLastError:     pgTextToStringPtr(r.ProxyLastError),
			CreatedAt:          r.CreatedAt,
		}
	}
	return servers, nil
}

// SetServerProxyReady marks proxy as successfully bootstrapped (mode=managed, clears error).
func SetServerProxyReady(ctx context.Context, serverID, status string) error {
	return Queries.SetServerProxyReady(ctx, sqlcgen.SetServerProxyReadyParams{
		ID:          serverID,
		ProxyStatus: status,
	})
}

// SetServerProxyError records a proxy failure without changing proxy_mode.
func SetServerProxyError(ctx context.Context, serverID, status string, lastError string) error {
	return Queries.SetServerProxyError(ctx, sqlcgen.SetServerProxyErrorParams{
		ID:             serverID,
		ProxyStatus:    status,
		ProxyLastError: pgtype.Text{String: lastError, Valid: true},
	})
}

type TLSPendingServer struct {
	ID            string
	Host          string
	Port          int32
	SSHUser       string
	EncryptedKey  string
}

func ListTLSPendingServers(ctx context.Context) ([]TLSPendingServer, error) {
	rows, err := Queries.ListTLSPendingServers(ctx)
	if err != nil {
		return nil, err
	}
	servers := make([]TLSPendingServer, len(rows))
	for i, r := range rows {
		servers[i] = TLSPendingServer{
			ID:           r.ID,
			Host:         r.Host,
			Port:         r.Port,
			SSHUser:      r.SshUser,
			EncryptedKey: r.PrivateKey,
		}
	}
	return servers, nil
}

func GetServerWithKey(ctx context.Context, id string) (ServerWithKey, error) {
	row, err := Queries.GetServerWithKey(ctx, id)
	if err != nil {
		return ServerWithKey{}, err
	}
	privateKey, err := crypto.Decrypt(row.PrivateKey)
	if err != nil {
		return ServerWithKey{}, fmt.Errorf("decrypt private key: %w", err)
	}
	return ServerWithKey{
		AppServer: AppServer{
			ID:          row.ID,
			Name:        row.Name,
			Host:        row.Host,
			Port:        row.Port,
			SSHUser:     row.SshUser,
			SSHKeyID:    row.SshKeyID,
			WorkspaceID: row.WorkspaceID,
			CreatedAt:   row.CreatedAt,
		},
		PrivateKey: privateKey,
	}, nil
}
