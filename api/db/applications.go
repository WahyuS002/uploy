package db

import (
	"context"
	"fmt"
	"time"

	"github.com/WahyuS002/uploy/crypto"
	"github.com/WahyuS002/uploy/db/sqlcgen"
	"github.com/jackc/pgx/v5/pgtype"
)

type Application struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Image         string    `json:"image"`
	ContainerName string    `json:"container_name"`
	Port          int32     `json:"port"`
	FQDN          *string   `json:"fqdn"`
	ServerID      string    `json:"server_id"`
	WorkspaceID   string    `json:"workspace_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ApplicationWithServer — dipakai saat deploy, satu query JOIN dapat semuanya
type ApplicationWithServer struct {
	Application
	Host           string `json:"-"`
	ServerPort     int32  `json:"-"`
	SSHUser        string `json:"-"`
	PrivateKey     string `json:"-"`
	ProxyInstalled bool   `json:"-"`
}

func pgTextToStringPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func stringPtrToPgText(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func CreateApplication(ctx context.Context, name, image, containerName string, port int32, serverID, workspaceID string, fqdn *string) (Application, error) {
	a, err := Queries.CreateApplication(ctx, sqlcgen.CreateApplicationParams{
		Name:          name,
		Image:         image,
		ContainerName: containerName,
		Port:          port,
		ServerID:      serverID,
		WorkspaceID:   workspaceID,
		Fqdn:          stringPtrToPgText(fqdn),
	})
	if err != nil {
		return Application{}, err
	}
	return Application{
		ID:            a.ID,
		Name:          a.Name,
		Image:         a.Image,
		ContainerName: a.ContainerName,
		Port:          a.Port,
		FQDN:          pgTextToStringPtr(a.Fqdn),
		ServerID:      a.ServerID,
		WorkspaceID:   a.WorkspaceID,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}, nil
}

func GetApplicationByID(ctx context.Context, id string) (Application, error) {
	a, err := Queries.GetApplicationByID(ctx, id)
	if err != nil {
		return Application{}, err
	}
	return Application{
		ID:            a.ID,
		Name:          a.Name,
		Image:         a.Image,
		ContainerName: a.ContainerName,
		Port:          a.Port,
		FQDN:          pgTextToStringPtr(a.Fqdn),
		ServerID:      a.ServerID,
		WorkspaceID:   a.WorkspaceID,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}, nil
}

func ListApplicationsByWorkspace(ctx context.Context, workspaceID string) ([]Application, error) {
	rows, err := Queries.ListApplicationsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	apps := make([]Application, len(rows))
	for i, r := range rows {
		apps[i] = Application{
			ID:            r.ID,
			Name:          r.Name,
			Image:         r.Image,
			ContainerName: r.ContainerName,
			Port:          r.Port,
			FQDN:          pgTextToStringPtr(r.Fqdn),
			ServerID:      r.ServerID,
			WorkspaceID:   r.WorkspaceID,
			CreatedAt:     r.CreatedAt,
			UpdatedAt:     r.UpdatedAt,
		}
	}
	return apps, nil
}

func UpdateApplication(ctx context.Context, id, name, image, containerName string, port int32, serverID string, fqdn *string) (Application, error) {
	a, err := Queries.UpdateApplication(ctx, sqlcgen.UpdateApplicationParams{
		ID:            id,
		Name:          name,
		Image:         image,
		ContainerName: containerName,
		Port:          port,
		ServerID:      serverID,
		Fqdn:          stringPtrToPgText(fqdn),
	})
	if err != nil {
		return Application{}, err
	}
	return Application{
		ID:            a.ID,
		Name:          a.Name,
		Image:         a.Image,
		ContainerName: a.ContainerName,
		Port:          a.Port,
		FQDN:          pgTextToStringPtr(a.Fqdn),
		ServerID:      a.ServerID,
		WorkspaceID:   a.WorkspaceID,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}, nil
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
			FQDN:          pgTextToStringPtr(row.Fqdn),
			ServerID:      row.ServerID,
			WorkspaceID:   row.WorkspaceID,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		},
		Host:           row.Host,
		ServerPort:     row.ServerPort,
		SSHUser:        row.SshUser,
		PrivateKey:     privateKey,
		ProxyInstalled: row.ProxyInstalled,
	}, nil
}
