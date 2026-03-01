package main

import (
	"fmt"
	"net/http"
	"os/exec"

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
	http.HandleFunc("/api/docker/ps", dockerPsHandler)
	http.HandleFunc("/api/docker/nginx", dockerNginxHandler)
	fmt.Println("Server berjalan di localhost:8080")
	http.ListenAndServe(":8080", nil)
}
