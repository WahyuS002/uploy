package db

import (
	"context"
	"fmt"
	"time"

	"github.com/WahyuS002/uploy/crypto"
	"github.com/WahyuS002/uploy/db/sqlcgen"
)

type Application struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Image         string    `json:"image"`
	ContainerName string    `json:"container_name"`
	Port          int32     `json:"port"`
	ServerID      string    `json:"server_id"`
	WorkspaceID   string    `json:"workspace_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ApplicationWithServer — dipakai saat deploy, satu query JOIN dapat semuanya
type ApplicationWithServer struct {
	Application
	Host       string `json:"-"`
	ServerPort int32  `json:"-"`
	SSHUser    string `json:"-"`
	PrivateKey string `json:"-"`
}

func applicationFromGen(a sqlcgen.Application) Application {
	return Application{
		ID:            a.ID,
		Name:          a.Name,
		Image:         a.Image,
		ContainerName: a.ContainerName,
		Port:          a.Port,
		ServerID:      a.ServerID,
		WorkspaceID:   a.WorkspaceID,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
}

func CreateApplication(ctx context.Context, name, image, containerName string, port int32, serverID, workspaceID string) (Application, error) {
	a, err := Queries.CreateApplication(ctx, sqlcgen.CreateApplicationParams{
		Name:          name,
		Image:         image,
		ContainerName: containerName,
		Port:          port,
		ServerID:      serverID,
		WorkspaceID:   workspaceID,
	})
	if err != nil {
		return Application{}, err
	}
	return applicationFromGen(a), nil
}

func GetApplicationByID(ctx context.Context, id string) (Application, error) {
	a, err := Queries.GetApplicationByID(ctx, id)
	if err != nil {
		return Application{}, err
	}
	return applicationFromGen(a), nil
}

func ListApplicationsByWorkspace(ctx context.Context, workspaceID string) ([]Application, error) {
	rows, err := Queries.ListApplicationsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	apps := make([]Application, len(rows))
	for i, r := range rows {
		apps[i] = applicationFromGen(r)
	}
	return apps, nil
}

func UpdateApplication(ctx context.Context, id, name, image, containerName string, port int32, serverID string) (Application, error) {
	a, err := Queries.UpdateApplication(ctx, sqlcgen.UpdateApplicationParams{
		ID:            id,
		Name:          name,
		Image:         image,
		ContainerName: containerName,
		Port:          port,
		ServerID:      serverID,
	})
	if err != nil {
		return Application{}, err
	}
	return applicationFromGen(a), nil
}

func DeleteApplication(ctx context.Context, id string) error {
	return Queries.DeleteApplication(ctx, id)
}

func GetApplicationWithServer(ctx context.Context, id string) (ApplicationWithServer, error) {
	row, err := Queries.GetApplicationWithServer(ctx, id)
	if err != nil {
		return ApplicationWithServer{}, err
	}
	privateKey, err := crypto.Decrypt(row.PrivateKey)
	if err != nil {
		return ApplicationWithServer{}, fmt.Errorf("decrypt private key: %w", err)
	}
	return ApplicationWithServer{
		Application: Application{
			ID:            row.ID,
			Name:          row.Name,
			Image:         row.Image,
			ContainerName: row.ContainerName,
			Port:          row.Port,
			ServerID:      row.ServerID,
			WorkspaceID:   row.WorkspaceID,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		},
		Host:       row.Host,
		ServerPort: row.ServerPort,
		SSHUser:    row.SshUser,
		PrivateKey: privateKey,
	}, nil
}
