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
	_, err := Pool.Exec(context.Background(),`
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
}