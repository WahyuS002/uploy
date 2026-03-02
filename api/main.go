package main

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/jobs"
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
	deploymentID := fmt.Sprintf("dep-%d", time.Now().UnixNano())
	_, err := db.Conn.Exec(context.Background(),
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
	db.Init()
	defer db.Conn.Close(context.Background())

	http.HandleFunc("/api/docker/ps", dockerPsHandler)
	http.HandleFunc("/api/docker/nginx", dockerNginxHandler)
	fmt.Println("Server berjalan di localhost:8080")
	http.ListenAndServe(":8080", nil)
}
