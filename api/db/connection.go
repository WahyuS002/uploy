package db

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func Init() {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://uploy:password@localhost:5432/uploy"
	}

	// 1) Parse config  for tuning
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		log.Fatal("Invalid DATABASE_URL: ", err)
	}

	// 2) Simple tuning, but useful
	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute

	// 3) Create pool
	Pool, err = pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		log.Fatal("Unable to create pool: ", err)
	}

	// 4) Fail-fast check
	if err := Pool.Ping(context.Background()); err != nil {
		log.Fatal("DB ping failed: ", err)
	}

	migrate()
}

func Close() {
	if Pool != nil {
		Pool.Close()
	}
}

func migrate() {
	_, err := Pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS deployments (
			id TEXT PRIMARY KEY,
			status TEXT
		);

		CREATE TABLE IF NOT EXISTS deployment_logs (
			id BIGSERIAL PRIMARY KEY,
			deployment_id TEXT NOT NULL REFERENCES deployments(id),
			output TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_deployment_logs_deployment_id_created_at
			ON deployment_logs (deployment_id, created_at)
	`)
	if err != nil {
		log.Fatal("Migrate failed: ", err)
	}

	_, err = Pool.Exec(context.Background(), `
		ALTER TABLE deployment_logs ADD COLUMN IF NOT EXISTS "order" INT;
		ALTER TABLE deployment_logs ADD COLUMN IF NOT EXISTS type TEXT NOT NULL DEFAULT 'stdout';
		CREATE INDEX IF NOT EXISTS idx_deployment_logs_deployment_id_order
			ON deployment_logs (deployment_id, "order");
	`)
	if err != nil {
		log.Fatal("Migrate failed: ", err)
	}

	_, err = Pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			platform_role TEXT NOT NULL DEFAULT 'user',
			status TEXT NOT NULL DEFAULT 'active',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);

		CREATE TABLE IF NOT EXISTS workspaces (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			owner_user_id TEXT NOT NULL REFERENCES users(id),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS workspace_memberships (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL REFERENCES workspaces(id),
			user_id TEXT NOT NULL REFERENCES users(id),
			role TEXT NOT NULL DEFAULT 'viewer',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_wm_workspace_user ON workspace_memberships (workspace_id, user_id);
		CREATE INDEX IF NOT EXISTS idx_wm_user_id ON workspace_memberships (user_id);

		CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id),
			workspace_id TEXT NOT NULL REFERENCES workspaces(id),
			workspace_role TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			expires_at TIMESTAMPTZ NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions (user_id);
		CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions (expires_at);

		ALTER TABLE deployments ADD COLUMN IF NOT EXISTS workspace_id TEXT REFERENCES workspaces(id);
	`)
	if err != nil {
		log.Fatal("Migrate failed: ", err)
	}
}
