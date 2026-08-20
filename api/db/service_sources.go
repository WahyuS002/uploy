package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/WahyuS002/uploy/db/sqlcgen"
	"github.com/jackc/pgx/v5/pgtype"
)

type ServiceSource struct {
	ServiceID string
	Provider  string
	Owner     string
	Repo      string
	Branch    string
	RootDir   *string
	Detected  json.RawMessage
	CreatedAt time.Time
	UpdatedAt time.Time
}

func serviceSourceFromRow(row sqlcgen.ServiceSource) ServiceSource {
	var rootDir *string
	if row.RootDir.Valid {
		value := row.RootDir.String
		rootDir = &value
	}
	return ServiceSource{
		ServiceID: row.ServiceID,
		Provider:  row.Provider,
		Owner:     row.Owner,
		Repo:      row.Repo,
		Branch:    row.Branch,
		RootDir:   rootDir,
		Detected:  json.RawMessage(row.Detected),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func GetServiceSource(ctx context.Context, serviceID string) (ServiceSource, error) {
	row, err := Queries.GetServiceSourceByServiceID(ctx, serviceID)
	if err != nil {
		return ServiceSource{}, err
	}
	return serviceSourceFromRow(row), nil
}

func ListServiceSources(ctx context.Context, serviceIDs []string) (map[string]ServiceSource, error) {
	result := make(map[string]ServiceSource, len(serviceIDs))
	if len(serviceIDs) == 0 {
		return result, nil
	}
	rows, err := Queries.ListServiceSourcesByServiceIDs(ctx, serviceIDs)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ServiceID] = serviceSourceFromRow(row)
	}
	return result, nil
}

// CreateServiceFromSource atomically creates the service and its one-to-one
// source record. The generated service id determines the stable image prefix.
func CreateServiceFromSource(
	ctx context.Context,
	name, containerName string,
	containerPort int32,
	hostPort *int32,
	serverID, workspaceID, projectID, environmentID string,
	provider, owner, repo, branch string,
	detected json.RawMessage,
) (Service, error) {
	tx, err := Pool.Begin(ctx)
	if err != nil {
		return Service{}, err
	}
	defer tx.Rollback(ctx)

	q := Queries.WithTx(tx)
	service, err := q.CreateSourceService(ctx, sqlcgen.CreateSourceServiceParams{
		Name:          name,
		ContainerName: containerName,
		ContainerPort: containerPort,
		HostPort:      pgInt4FromInt32Ptr(hostPort),
		ServerID:      serverID,
		WorkspaceID:   workspaceID,
		Kind:          "application",
		ProjectID:     projectID,
		EnvironmentID: environmentID,
	})
	if err != nil {
		return Service{}, err
	}

	_, err = q.CreateServiceSource(ctx, sqlcgen.CreateServiceSourceParams{
		ServiceID: service.ID,
		Provider:  provider,
		Owner:     owner,
		Repo:      repo,
		Branch:    branch,
		RootDir:   pgtype.Text{},
		Detected:  detected,
	})
	if err != nil {
		return Service{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Service{}, err
	}

	return Service{
		ID: service.ID, Name: service.Name, Image: service.Image,
		ContainerName: service.ContainerName, ContainerPort: service.ContainerPort,
		HostPort: int32PtrFromPgInt4(service.HostPort), ServerID: service.ServerID,
		WorkspaceID: service.WorkspaceID, Kind: service.Kind,
		ProjectID: service.ProjectID, EnvironmentID: service.EnvironmentID,
		CreatedAt: service.CreatedAt, UpdatedAt: service.UpdatedAt,
		HasDeployed: service.HasDeployed,
	}, nil
}
