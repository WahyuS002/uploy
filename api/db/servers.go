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
	ID                    string           `json:"id"`
	Name                  string           `json:"name"`
	Host                  string           `json:"host"`
	Port                  int32            `json:"port"`
	SSHUser               string           `json:"ssh_user"`
	SSHKeyID              string           `json:"ssh_key_id"`
	WorkspaceID           string           `json:"workspace_id"`
	ProxyStatus           string           `json:"proxy_status"`
	ProxyLastReconciledAt *time.Time       `json:"proxy_last_reconciled_at"`
	ProxyLastError        *string          `json:"proxy_last_error"`
	CreatedAt             time.Time        `json:"created_at"`
	Monitoring            ServerMonitoring `json:"monitoring"`
}

type ServerMonitoring struct {
	Enabled          bool       `json:"enabled"`
	Port             int32      `json:"port"`
	RetentionDays    int32      `json:"retention_days"`
	FQDN             *string    `json:"fqdn"`
	Status           string     `json:"status"`
	LastReconciledAt *time.Time `json:"last_reconciled_at"`
	LastError        *string    `json:"last_error"`
	CleanupAt        *time.Time `json:"cleanup_at"`
}

type ServerWithKey struct {
	AppServer
	PrivateKey   string `json:"-"`
	ControlToken string `json:"-"`
	ReaderToken  string `json:"-"`
}

type MonitoringConfig struct {
	Enabled       bool
	Port          int32
	RetentionDays int32
	FQDN          string
	Status        string
	LastError     string
	CleanupAt     *time.Time
}

func CreateServer(ctx context.Context, name, host string, port int32, sshUser, sshKeyID, workspaceID string) (AppServer, error) {
	row, err := Queries.CreateServer(ctx, sqlcgen.CreateServerParams{
		Name: name, Host: host, Port: port, SshUser: sshUser, SshKeyID: sshKeyID, WorkspaceID: workspaceID,
	})
	if err != nil {
		return AppServer{}, err
	}
	return newAppServer(
		row.ID, row.Name, row.Host, row.Port, row.SshUser, row.SshKeyID, row.WorkspaceID,
		row.ProxyStatus, row.ProxyLastReconciledAt, row.ProxyLastError, row.CreatedAt,
		row.MonitoringEnabled, row.MonitoringPort, row.MonitoringRetentionDays,
		row.MonitoringFqdn, row.MonitoringStatus, row.MonitoringLastReconciledAt, row.MonitoringLastError, row.MonitoringCleanupAt,
	), nil
}

func GetServerByID(ctx context.Context, id string) (AppServer, error) {
	row, err := Queries.GetServerByID(ctx, id)
	if err != nil {
		return AppServer{}, err
	}
	return newAppServer(
		row.ID, row.Name, row.Host, row.Port, row.SshUser, row.SshKeyID, row.WorkspaceID,
		row.ProxyStatus, row.ProxyLastReconciledAt, row.ProxyLastError, row.CreatedAt,
		row.MonitoringEnabled, row.MonitoringPort, row.MonitoringRetentionDays,
		row.MonitoringFqdn, row.MonitoringStatus, row.MonitoringLastReconciledAt, row.MonitoringLastError, row.MonitoringCleanupAt,
	), nil
}

func ListServersByWorkspace(ctx context.Context, workspaceID string) ([]AppServer, error) {
	rows, err := Queries.ListServersByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	servers := make([]AppServer, len(rows))
	for index, row := range rows {
		servers[index] = newAppServer(
			row.ID, row.Name, row.Host, row.Port, row.SshUser, row.SshKeyID, row.WorkspaceID,
			row.ProxyStatus, row.ProxyLastReconciledAt, row.ProxyLastError, row.CreatedAt,
			row.MonitoringEnabled, row.MonitoringPort, row.MonitoringRetentionDays,
			row.MonitoringFqdn, row.MonitoringStatus, row.MonitoringLastReconciledAt, row.MonitoringLastError, row.MonitoringCleanupAt,
		)
	}
	return servers, nil
}

func SetServerProxyReady(ctx context.Context, serverID, status string) error {
	return Queries.SetServerProxyReady(ctx, sqlcgen.SetServerProxyReadyParams{ID: serverID, ProxyStatus: status})
}

func SetServerProxyError(ctx context.Context, serverID, status, lastError string) error {
	return Queries.SetServerProxyError(ctx, sqlcgen.SetServerProxyErrorParams{
		ID: serverID, ProxyStatus: status, ProxyLastError: pgtype.Text{String: lastError, Valid: true},
	})
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
	controlToken, err := decryptOptional(row.MonitoringControlToken)
	if err != nil {
		return ServerWithKey{}, fmt.Errorf("decrypt monitoring control token: %w", err)
	}
	readerToken, err := decryptOptional(row.MonitoringReaderToken)
	if err != nil {
		return ServerWithKey{}, fmt.Errorf("decrypt monitoring reader token: %w", err)
	}
	return ServerWithKey{
		AppServer: newAppServer(
			row.ID, row.Name, row.Host, row.Port, row.SshUser, row.SshKeyID, row.WorkspaceID,
			"", pgtype.Timestamptz{}, pgtype.Text{}, row.CreatedAt,
			row.MonitoringEnabled, row.MonitoringPort, row.MonitoringRetentionDays,
			row.MonitoringFqdn, row.MonitoringStatus, row.MonitoringLastReconciledAt, row.MonitoringLastError, row.MonitoringCleanupAt,
		),
		PrivateKey: privateKey, ControlToken: controlToken, ReaderToken: readerToken,
	}, nil
}

func SetServerMonitoring(ctx context.Context, serverID string, config MonitoringConfig, controlToken, readerToken string) error {
	controlCiphertext, err := encryptOptional(controlToken)
	if err != nil {
		return fmt.Errorf("encrypt monitoring control token: %w", err)
	}
	readerCiphertext, err := encryptOptional(readerToken)
	if err != nil {
		return fmt.Errorf("encrypt monitoring reader token: %w", err)
	}
	cleanupAt := time.Unix(0, 0).UTC()
	if config.CleanupAt != nil {
		cleanupAt = config.CleanupAt.UTC()
	}
	return Queries.SetServerMonitoring(ctx, sqlcgen.SetServerMonitoringParams{
		MonitoringEnabled: config.Enabled, MonitoringPort: config.Port, MonitoringRetentionDays: config.RetentionDays,
		MonitoringFqdn:         config.FQDN,
		MonitoringControlToken: controlCiphertext, MonitoringReaderToken: readerCiphertext,
		MonitoringStatus: config.Status, MonitoringLastError: config.LastError, MonitoringCleanupAt: cleanupAt, ID: serverID,
	})
}

func MonitoringFQDNInUse(ctx context.Context, serverID, fqdn string) (bool, error) {
	return Queries.ServerFQDNInUse(ctx, sqlcgen.ServerFQDNInUseParams{ID: serverID, Fqdn: fqdn})
}

func ListMonitoringCleanupDue(ctx context.Context) ([]ServerWithKey, error) {
	rows, err := Queries.ListMonitoringCleanupDue(ctx)
	if err != nil {
		return nil, err
	}
	servers := make([]ServerWithKey, 0, len(rows))
	for _, row := range rows {
		privateKey, decryptErr := crypto.Decrypt(row.PrivateKey)
		if decryptErr != nil {
			return nil, fmt.Errorf("decrypt SSH key for server %s: %w", row.ID, decryptErr)
		}
		servers = append(servers, ServerWithKey{
			AppServer: newAppServer(
				row.ID, row.Name, row.Host, row.Port, row.SshUser, row.SshKeyID, row.WorkspaceID,
				"", pgtype.Timestamptz{}, pgtype.Text{}, row.CreatedAt,
				row.MonitoringEnabled, row.MonitoringPort, row.MonitoringRetentionDays,
				row.MonitoringFqdn, row.MonitoringStatus, row.MonitoringLastReconciledAt, row.MonitoringLastError, row.MonitoringCleanupAt,
			),
			PrivateKey: privateKey,
		})
	}
	return servers, nil
}

func ClearServerMonitoringData(ctx context.Context, serverID string) error {
	return Queries.ClearServerMonitoringData(ctx, serverID)
}

func newAppServer(id, name, host string, port int32, sshUser, sshKeyID, workspaceID, proxyStatus string,
	proxyReconciledAt pgtype.Timestamptz, proxyLastError pgtype.Text, createdAt time.Time,
	monitoringEnabled bool, monitoringPort, retentionDays int32, fqdn pgtype.Text,
	monitoringStatus string, monitoringReconciledAt pgtype.Timestamptz, monitoringLastError pgtype.Text,
	monitoringCleanupAt pgtype.Timestamptz) AppServer {
	return AppServer{
		ID: id, Name: name, Host: host, Port: port, SSHUser: sshUser, SSHKeyID: sshKeyID, WorkspaceID: workspaceID,
		ProxyStatus: proxyStatus, ProxyLastReconciledAt: timePtrFromPgTimestamptz(proxyReconciledAt),
		ProxyLastError: stringPtrFromPgText(proxyLastError), CreatedAt: createdAt,
		Monitoring: ServerMonitoring{
			Enabled: monitoringEnabled, Port: monitoringPort, RetentionDays: retentionDays,
			FQDN: stringPtrFromPgText(fqdn), Status: monitoringStatus,
			LastReconciledAt: timePtrFromPgTimestamptz(monitoringReconciledAt),
			LastError:        stringPtrFromPgText(monitoringLastError), CleanupAt: timePtrFromPgTimestamptz(monitoringCleanupAt),
		},
	}
}

func encryptOptional(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	return crypto.Encrypt(value)
}

func decryptOptional(value pgtype.Text) (string, error) {
	if !value.Valid || value.String == "" {
		return "", nil
	}
	return crypto.Decrypt(value.String)
}
