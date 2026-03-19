package db

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func Init(databaseURL string) {
	// 1) Parse config  for tuning
	cfg, err := pgxpool.ParseConfig(databaseURL)
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
}

func Close() {
	if Pool != nil {
		Pool.Close()
	}
}
