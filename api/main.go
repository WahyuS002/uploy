package main

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"

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
	deployment, err := db.CreateDeployment(context.Background())

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	go jobs.RunNginx(deployment.ID)

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
