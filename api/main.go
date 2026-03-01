package main

import (
	"fmt"
	"net/http"
	"os/exec"
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
	out, err := exec.Command("docker", "pull", "nginx:latest").Output()

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write(out)
}

func main() {
	http.HandleFunc("/api/docker/ps", dockerPsHandler)
	http.HandleFunc("/api/docker/nginx", dockerNginxHandler)
	fmt.Println("Server berjalan di localhost:8080")
	http.ListenAndServe(":8080", nil)
}
