package db

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

var Conn *pgx.Conn

func Init() {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://uploy:password@localhost:5432/uploy"
	}

	var err error
	Conn, err = pgx.Connect(context.Background(), url)
	if err != nil {
		log.Fatal("Unable to connect to database: ", err)
	}

	migrate()
}

func migrate() {
	_, err := Conn.Exec(context.Background(),
		`CREATE TABLE IF NOT EXISTS deployments (
			id TEXT PRIMARY KEY,
			status TEXT
		)`)

	if err != nil {
		log.Fatal("migrate failed: ", err)
	}
}
