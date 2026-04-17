package db

import (
	"context"
	"fmt"
	"time"

	"github.com/WahyuS002/uploy/crypto"
	"github.com/WahyuS002/uploy/db/sqlcgen"
)

type Service struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Image         string    `json:"image"`
	ContainerName string    `json:"container_name"`
	Port          int32     `json:"port"`
	ServerID      string    `json:"server_id"`
	WorkspaceID   string    `json:"workspace_id"`
	Kind          string    `json:"kind"`
	ProjectID     string    `json:"project_id"`
	EnvironmentID string    `json:"environment_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ServiceWithServer is used during deploy — one JOIN query gets everything.
type ServiceWithServer struct {
	Service
	Host        string `json:"-"`
	ServerPort  int32  `json:"-"`
	SSHUser     string `json:"-"`
	PrivateKey  string `json:"-"`
	ProxyStatus string `json:"-"`
}

func CreateService(ctx context.Context, name, image, containerName string, port int32, serverID, workspaceID, kind, projectID, environmentID string) (Service, error) {
	r, err := Queries.CreateService(ctx, sqlcgen.CreateServiceParams{
		Name:          name,
		Image:         image,
		ContainerName: containerName,
		Port:          port,
		ServerID:      serverID,
		WorkspaceID:   workspaceID,
		Kind:          kind,
		ProjectID:     projectID,
		EnvironmentID: environmentID,
	})
	if err != nil {
		return Service{}, err
	}
	return Service{
		ID: r.ID, Name: r.Name, Image: r.Image, ContainerName: r.ContainerName,
		Port: r.Port, ServerID: r.ServerID, WorkspaceID: r.WorkspaceID,
		Kind: r.Kind, ProjectID: r.ProjectID, EnvironmentID: r.EnvironmentID,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, nil
}

func GetServiceByID(ctx context.Context, id string) (Service, error) {
	r, err := Queries.GetServiceByID(ctx, id)
	if err != nil {
		return Service{}, err
	}
	return Service{
		ID: r.ID, Name: r.Name, Image: r.Image, ContainerName: r.ContainerName,
		Port: r.Port, ServerID: r.ServerID, WorkspaceID: r.WorkspaceID,
		Kind: r.Kind, ProjectID: r.ProjectID, EnvironmentID: r.EnvironmentID,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, nil
}

func ListServicesByWorkspace(ctx context.Context, workspaceID string) ([]Service, error) {
	rows, err := Queries.ListServicesByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	services := make([]Service, len(rows))
	for i, r := range rows {
		services[i] = Service{
			ID: r.ID, Name: r.Name, Image: r.Image, ContainerName: r.ContainerName,
			Port: r.Port, ServerID: r.ServerID, WorkspaceID: r.WorkspaceID,
			Kind: r.Kind, ProjectID: r.ProjectID, EnvironmentID: r.EnvironmentID,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}
	}
	return services, nil
}

func ListServicesByEnvironment(ctx context.Context, environmentID string) ([]Service, error) {
	rows, err := Queries.ListServicesByEnvironment(ctx, environmentID)
	if err != nil {
		return nil, err
	}
	services := make([]Service, len(rows))
	for i, r := range rows {
		services[i] = Service{
			ID: r.ID, Name: r.Name, Image: r.Image, ContainerName: r.ContainerName,
			Port: r.Port, ServerID: r.ServerID, WorkspaceID: r.WorkspaceID,
			Kind: r.Kind, ProjectID: r.ProjectID, EnvironmentID: r.EnvironmentID,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}
	}
	return services, nil
}

func ListServicesByProject(ctx context.Context, projectID string) ([]Service, error) {
	rows, err := Queries.ListServicesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	services := make([]Service, len(rows))
	for i, r := range rows {
		services[i] = Service{
			ID: r.ID, Name: r.Name, Image: r.Image, ContainerName: r.ContainerName,
			Port: r.Port, ServerID: r.ServerID, WorkspaceID: r.WorkspaceID,
			Kind: r.Kind, ProjectID: r.ProjectID, EnvironmentID: r.EnvironmentID,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}
	}
	return services, nil
}

func UpdateService(ctx context.Context, id, name, image, containerName string, port int32, serverID string) (Service, error) {
	r, err := Queries.UpdateService(ctx, sqlcgen.UpdateServiceParams{
		ID:            id,
		Name:          name,
		Image:         image,
		ContainerName: containerName,
		Port:          port,
		ServerID:      serverID,
	})
	if err != nil {
		return Service{}, err
	}
	return Service{
		ID: r.ID, Name: r.Name, Image: r.Image, ContainerName: r.ContainerName,
		Port: r.Port, ServerID: r.ServerID, WorkspaceID: r.WorkspaceID,
		Kind: r.Kind, ProjectID: r.ProjectID, EnvironmentID: r.EnvironmentID,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, nil
}

func DeleteService(ctx context.Context, id string) error {
	return Queries.DeleteService(ctx, id)
}

func GetServiceWithServer(ctx context.Context, id string) (ServiceWithServer, error) {
	row, err := Queries.GetServiceWithServer(ctx, id)
	if err != nil {
		return ServiceWithServer{}, err
	}
	privateKey, err := crypto.Decrypt(row.PrivateKey)
	if err != nil {
		return ServiceWithServer{}, fmt.Errorf("decrypt private key: %w", err)
	}
	return ServiceWithServer{
		Service: Service{
			ID: row.ID, Name: row.Name, Image: row.Image, ContainerName: row.ContainerName,
			Port: row.Port, ServerID: row.ServerID, WorkspaceID: row.WorkspaceID,
			Kind: row.Kind, ProjectID: row.ProjectID, EnvironmentID: row.EnvironmentID,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		},
		Host:        row.Host,
		ServerPort:  row.ServerPort,
		SSHUser:     row.SshUser,
		PrivateKey:  privateKey,
		ProxyStatus: row.ProxyStatus,
	}, nil
}
