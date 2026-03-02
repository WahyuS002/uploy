package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"github.com/WahyuS002/uploy/jobs"
	"github.com/jackc/pgx/v5"
)

func dockerPsHandler(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("docker", "ps").Output()

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write(out)
}

func dockerNginxHandler(w http.ResponseWriter, r *http.Request) {
	// out, err := exec.Command("docker", "pull", "nginx:latest").Output()

	go jobs.RunNginx()

	// if err != nil {
	// 	http.Error(w, err.Error(), 500)
	// 	return
	// }

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "success"}`))
}

func main() {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://uploy:password@localhost:5432/uploy"
	}

	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	http.HandleFunc("/api/docker/ps", dockerPsHandler)
	http.HandleFunc("/api/docker/nginx", dockerNginxHandler)
	fmt.Println("Server berjalan di localhost:8080")
	http.ListenAndServe(":8080", nil)
}
