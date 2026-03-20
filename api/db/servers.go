package db

import (
	"context"
	"time"

	"github.com/WahyuS002/uploy/db/sqlcgen"
)

type AppServer struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Host        string    `json:"host"`
	Port        int32     `json:"port"`
	SSHUser     string    `json:"ssh_user"`
	SSHKeyID    string    `json:"ssh_key_id"`
	WorkspaceID string    `json:"workspace_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type ServerWithKey struct {
	AppServer
	PrivateKey string `json:"-"`
}

func serverFromGen(s sqlcgen.Server) AppServer {
	return AppServer{
		ID:          s.ID,
		Name:        s.Name,
		Host:        s.Host,
		Port:        s.Port,
		SSHUser:     s.SshUser,
		SSHKeyID:    s.SshKeyID,
		WorkspaceID: s.WorkspaceID,
		CreatedAt:   s.CreatedAt,
	}
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
	return serverFromGen(s), nil
}

func GetServerByID(ctx context.Context, id string) (AppServer, error) {
	s, err := Queries.GetServerByID(ctx, id)
	if err != nil {
		return AppServer{}, err
	}
	return serverFromGen(s), nil
}

func ListServersByWorkspace(ctx context.Context, workspaceID string) ([]AppServer, error) {
	rows, err := Queries.ListServersByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	servers := make([]AppServer, len(rows))
	for i, r := range rows {
		servers[i] = serverFromGen(r)
	}
	return servers, nil
}

func GetServerWithKey(ctx context.Context, id string) (ServerWithKey, error) {
	row, err := Queries.GetServerWithKey(ctx, id)
	if err != nil {
		return ServerWithKey{}, err
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
		PrivateKey: row.PrivateKey,
	}, nil
}
