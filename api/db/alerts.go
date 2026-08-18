package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/WahyuS002/uploy/crypto"
	"github.com/WahyuS002/uploy/db/sqlcgen"
	"github.com/jackc/pgx/v5/pgtype"
)

type NotificationChannel struct {
	ID          string                 `json:"id"`
	WorkspaceID string                 `json:"workspace_id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type AlertRule struct {
	ID              string    `json:"id"`
	WorkspaceID     string    `json:"workspace_id"`
	Name            string    `json:"name"`
	Condition       string    `json:"condition"`
	Threshold       float64   `json:"threshold"`
	DurationSeconds int32     `json:"duration_seconds"`
	ScopeType       string    `json:"scope_type"`
	ServerID        *string   `json:"server_id,omitempty"`
	ServiceID       *string   `json:"service_id,omitempty"`
	Enabled         bool      `json:"enabled"`
	ChannelIDs      []string  `json:"channel_ids"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type AlertEvent struct {
	ID            string     `json:"id"`
	WorkspaceID   string     `json:"workspace_id"`
	RuleID        string     `json:"rule_id"`
	TargetID      string     `json:"target_id"`
	TargetName    string     `json:"target_name"`
	Status        string     `json:"status"`
	StartedAt     time.Time  `json:"started_at"`
	ResolvedAt    *time.Time `json:"resolved_at,omitempty"`
	TriggerValue  float64    `json:"trigger_value"`
	ResolvedValue *float64   `json:"resolved_value,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func CreateNotificationChannel(ctx context.Context, workspaceID, name, kind string, config map[string]interface{}) (NotificationChannel, error) {
	ciphertext, err := encodeChannelConfig(config)
	if err != nil {
		return NotificationChannel{}, err
	}
	row, err := Queries.CreateNotificationChannel(ctx, sqlcgen.CreateNotificationChannelParams{
		WorkspaceID: workspaceID, Name: name, Type: kind, ConfigCiphertext: ciphertext,
	})
	if err != nil {
		return NotificationChannel{}, err
	}
	return notificationChannelFromRow(row)
}

func ListNotificationChannels(ctx context.Context, workspaceID string) ([]NotificationChannel, error) {
	rows, err := Queries.ListNotificationChannels(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	channels := make([]NotificationChannel, 0, len(rows))
	for _, row := range rows {
		channel, err := notificationChannelFromRow(row)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}
	return channels, nil
}

func GetNotificationChannel(ctx context.Context, id string) (NotificationChannel, error) {
	row, err := Queries.GetNotificationChannel(ctx, id)
	if err != nil {
		return NotificationChannel{}, err
	}
	return notificationChannelFromRow(row)
}

func UpdateNotificationChannel(ctx context.Context, id, name, kind string, config map[string]interface{}, enabled bool) (NotificationChannel, error) {
	ciphertext, err := encodeChannelConfig(config)
	if err != nil {
		return NotificationChannel{}, err
	}
	row, err := Queries.UpdateNotificationChannel(ctx, sqlcgen.UpdateNotificationChannelParams{
		ID: id, Name: name, Type: kind, ConfigCiphertext: ciphertext, Enabled: enabled,
	})
	if err != nil {
		return NotificationChannel{}, err
	}
	return notificationChannelFromRow(row)
}

func DeleteNotificationChannel(ctx context.Context, id string) error {
	return Queries.DeleteNotificationChannel(ctx, id)
}

func CreateAlertRule(ctx context.Context, workspaceID, name, condition string, threshold float64, durationSeconds int32, scopeType string, serverID, serviceID *string, channelIDs []string) (AlertRule, error) {
	row, err := Queries.CreateAlertRule(ctx, sqlcgen.CreateAlertRuleParams{
		WorkspaceID: workspaceID, Name: name, Condition: condition, Threshold: threshold,
		DurationSeconds: durationSeconds, ScopeType: scopeType, ServerID: nullableText(serverID), ServiceID: nullableText(serviceID),
	})
	if err != nil {
		return AlertRule{}, err
	}
	if err := replaceAlertRuleChannels(ctx, row.ID, channelIDs); err != nil {
		return AlertRule{}, err
	}
	return alertRuleFromRow(ctx, row)
}

func ListAlertRules(ctx context.Context, workspaceID string) ([]AlertRule, error) {
	rows, err := Queries.ListAlertRules(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	rules := make([]AlertRule, 0, len(rows))
	for _, row := range rows {
		rule, err := alertRuleFromRow(ctx, row)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func ListEnabledAlertRules(ctx context.Context) ([]AlertRule, error) {
	rows, err := Queries.ListEnabledAlertRules(ctx)
	if err != nil {
		return nil, err
	}
	rules := make([]AlertRule, 0, len(rows))
	for _, row := range rows {
		rule, err := alertRuleFromRow(ctx, row)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func GetAlertRule(ctx context.Context, id string) (AlertRule, error) {
	row, err := Queries.GetAlertRule(ctx, id)
	if err != nil {
		return AlertRule{}, err
	}
	return alertRuleFromRow(ctx, row)
}

func UpdateAlertRule(ctx context.Context, id, name, condition string, threshold float64, durationSeconds int32, scopeType string, serverID, serviceID *string, enabled bool, channelIDs []string) (AlertRule, error) {
	row, err := Queries.UpdateAlertRule(ctx, sqlcgen.UpdateAlertRuleParams{
		ID: id, Name: name, Condition: condition, Threshold: threshold, DurationSeconds: durationSeconds,
		ScopeType: scopeType, ServerID: nullableText(serverID), ServiceID: nullableText(serviceID), Enabled: enabled,
	})
	if err != nil {
		return AlertRule{}, err
	}
	if err := replaceAlertRuleChannels(ctx, id, channelIDs); err != nil {
		return AlertRule{}, err
	}
	return alertRuleFromRow(ctx, row)
}

func DeleteAlertRule(ctx context.Context, id string) error {
	return Queries.DeleteAlertRule(ctx, id)
}

func CreateAlertEvent(ctx context.Context, workspaceID, ruleID, targetID, targetName string, startedAt time.Time, value float64) (AlertEvent, error) {
	row, err := Queries.CreateAlertEvent(ctx, sqlcgen.CreateAlertEventParams{
		WorkspaceID: workspaceID, RuleID: ruleID, TargetID: targetID, TargetName: targetName, StartedAt: startedAt, TriggerValue: value,
	})
	if err != nil {
		return AlertEvent{}, err
	}
	return alertEventFromRow(row), nil
}

func FindActiveAlertEvent(ctx context.Context, ruleID, targetID string) (AlertEvent, error) {
	row, err := Queries.FindActiveAlertEvent(ctx, sqlcgen.FindActiveAlertEventParams{RuleID: ruleID, TargetID: targetID})
	if err != nil {
		return AlertEvent{}, err
	}
	return alertEventFromRow(row), nil
}

func ResolveAlertEvent(ctx context.Context, id string, resolvedAt time.Time, value float64) (AlertEvent, error) {
	row, err := Queries.ResolveAlertEvent(ctx, sqlcgen.ResolveAlertEventParams{ID: id, ResolvedAt: pgtype.Timestamptz{Time: resolvedAt, Valid: true}, ResolvedValue: pgtype.Float8{Float64: value, Valid: true}})
	if err != nil {
		return AlertEvent{}, err
	}
	return alertEventFromRow(row), nil
}

func ListAlertEvents(ctx context.Context, workspaceID string, limit, offset int32) ([]AlertEvent, error) {
	rows, err := Queries.ListAlertEvents(ctx, sqlcgen.ListAlertEventsParams{WorkspaceID: workspaceID, Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	events := make([]AlertEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, alertEventFromRow(row))
	}
	return events, nil
}

func ChannelConfig(ctx context.Context, id string) (NotificationChannel, map[string]interface{}, error) {
	channel, err := GetNotificationChannel(ctx, id)
	if err != nil {
		return NotificationChannel{}, nil, err
	}
	return channel, channel.Config, nil
}

func notificationChannelFromRow(row sqlcgen.NotificationChannel) (NotificationChannel, error) {
	config, err := decodeChannelConfig(row.ConfigCiphertext)
	if err != nil {
		return NotificationChannel{}, err
	}
	return NotificationChannel{ID: row.ID, WorkspaceID: row.WorkspaceID, Name: row.Name, Type: row.Type, Config: config, Enabled: row.Enabled, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func alertRuleFromRow(ctx context.Context, row sqlcgen.AlertRule) (AlertRule, error) {
	channels, err := Queries.ListAlertRuleChannels(ctx, row.ID)
	if err != nil {
		return AlertRule{}, err
	}
	ids := make([]string, 0, len(channels))
	for _, channel := range channels {
		ids = append(ids, channel)
	}
	return AlertRule{ID: row.ID, WorkspaceID: row.WorkspaceID, Name: row.Name, Condition: row.Condition, Threshold: row.Threshold, DurationSeconds: row.DurationSeconds, ScopeType: row.ScopeType, ServerID: pgTextPtr(row.ServerID), ServiceID: pgTextPtr(row.ServiceID), Enabled: row.Enabled, ChannelIDs: ids, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func alertEventFromRow(row sqlcgen.AlertEvent) AlertEvent {
	var resolvedAt *time.Time
	if row.ResolvedAt.Valid {
		value := row.ResolvedAt.Time
		resolvedAt = &value
	}
	var resolvedValue *float64
	if row.ResolvedValue.Valid {
		value := row.ResolvedValue.Float64
		resolvedValue = &value
	}
	return AlertEvent{ID: row.ID, WorkspaceID: row.WorkspaceID, RuleID: row.RuleID, TargetID: row.TargetID, TargetName: row.TargetName, Status: row.Status, StartedAt: row.StartedAt, ResolvedAt: resolvedAt, TriggerValue: row.TriggerValue, ResolvedValue: resolvedValue, CreatedAt: row.CreatedAt}
}

func replaceAlertRuleChannels(ctx context.Context, ruleID string, channelIDs []string) error {
	if err := Queries.DeleteAlertRuleChannels(ctx, ruleID); err != nil {
		return err
	}
	for _, channelID := range channelIDs {
		if err := Queries.AddAlertRuleChannel(ctx, sqlcgen.AddAlertRuleChannelParams{RuleID: ruleID, ChannelID: channelID}); err != nil {
			return err
		}
	}
	return nil
}

func encodeChannelConfig(value map[string]interface{}) (string, error) {
	if value == nil {
		value = map[string]interface{}{}
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal notification config: %w", err)
	}
	encoded, err := crypto.Encrypt(string(data))
	if err != nil {
		return "", fmt.Errorf("encrypt notification config: %w", err)
	}
	return encoded, nil
}

func decodeChannelConfig(value string) (map[string]interface{}, error) {
	plaintext, err := crypto.Decrypt(value)
	if err != nil {
		return nil, fmt.Errorf("decrypt notification config: %w", err)
	}
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(plaintext), &config); err != nil {
		return nil, fmt.Errorf("decode notification config: %w", err)
	}
	return config, nil
}

func nullableText(value *string) pgtype.Text {
	if value == nil || *value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}

func pgTextPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	copy := value.String
	return &copy
}
