-- +goose Up

CREATE TABLE notification_channels (
    id TEXT PRIMARY KEY DEFAULT 'chn-' || gen_random_uuid()::text,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('discord', 'slack', 'telegram', 'email', 'webhook')),
    config_ciphertext TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id, name)
);
CREATE INDEX idx_notification_channels_workspace ON notification_channels(workspace_id);

CREATE TABLE alert_rules (
    id TEXT PRIMARY KEY DEFAULT 'alr-' || gen_random_uuid()::text,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    condition TEXT NOT NULL CHECK (condition IN ('cpu_high', 'memory_high', 'disk_low', 'service_down', 'server_unreachable')),
    threshold DOUBLE PRECISION NOT NULL DEFAULT 0,
    duration_seconds INTEGER NOT NULL DEFAULT 300 CHECK (duration_seconds BETWEEN 60 AND 2592000),
    scope_type TEXT NOT NULL CHECK (scope_type IN ('server', 'service')),
    server_id TEXT REFERENCES servers(id) ON DELETE CASCADE,
    service_id TEXT REFERENCES services(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((scope_type = 'server' AND server_id IS NOT NULL AND service_id IS NULL)
        OR (scope_type = 'service' AND service_id IS NOT NULL AND server_id IS NULL))
);
CREATE INDEX idx_alert_rules_workspace ON alert_rules(workspace_id);
CREATE INDEX idx_alert_rules_enabled ON alert_rules(enabled);

CREATE TABLE alert_rule_channels (
    rule_id TEXT NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    channel_id TEXT NOT NULL REFERENCES notification_channels(id) ON DELETE CASCADE,
    PRIMARY KEY (rule_id, channel_id)
);

CREATE TABLE alert_events (
    id TEXT PRIMARY KEY DEFAULT 'evt-' || gen_random_uuid()::text,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    rule_id TEXT NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    target_id TEXT NOT NULL,
    target_name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('firing', 'resolved')),
    started_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ,
    trigger_value DOUBLE PRECISION NOT NULL DEFAULT 0,
    resolved_value DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_alert_events_workspace_started ON alert_events(workspace_id, started_at DESC);
CREATE INDEX idx_alert_events_rule_target ON alert_events(rule_id, target_id, started_at DESC);
CREATE UNIQUE INDEX idx_alert_events_active ON alert_events(rule_id, target_id) WHERE status = 'firing';

-- +goose Down

DROP TABLE IF EXISTS alert_events;
DROP TABLE IF EXISTS alert_rule_channels;
DROP TABLE IF EXISTS alert_rules;
DROP TABLE IF EXISTS notification_channels;
