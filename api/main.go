package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/WahyuS002/uploy/jobs"
	"github.com/jackc/pgx/v5"
)

var conn *pgx.Conn

func dockerPsHandler(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("docker", "ps").Output()

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write(out)
}

func dockerNginxHandler(w http.ResponseWriter, r *http.Request) {
	deploymentID := fmt.Sprintf("dep-%d", time.Now().UnixNano())
	_, err := conn.Exec(context.Background(),
		`INSERT INTO deployments (id, status) VALUES ($1, 'in_progress')`, deploymentID)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	go jobs.RunNginx()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "success"}`))
}

func main() {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://uploy:password@localhost:5432/uploy"
	}

	var err error

	conn, err = pgx.Connect(context.Background(), url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	conn.Exec(context.Background(),
		`CREATE TABLE IF NOT EXISTS deployments (
			id TEXT PRIMARY KEY,
			status TEXT
		)`)

	http.HandleFunc("/api/docker/ps", dockerPsHandler)
	http.HandleFunc("/api/docker/nginx", dockerNginxHandler)
	fmt.Println("Server berjalan di localhost:8080")
	http.ListenAndServe(":8080", nil)
}
