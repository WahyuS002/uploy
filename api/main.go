package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/handlers"
	"github.com/WahyuS002/uploy/jobs"
	"github.com/WahyuS002/uploy/ssh"
)

func dockerPsHandler(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("docker", "ps").Output()

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write(out)
}

func deployHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Image         string `json:"image"`
		ContainerName string `json:"container_name"`
		Port          int    `json:"port"`
		Server        struct {
			Host       string `json:"host"`
			Port       int    `json:"port"`
			User       string `json:"user"`
			PrivateKey string `json:"private_key"`
		} `json:"server"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", 400)
		return
	}

	deployment, err := db.CreateDeployment(context.Background())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	go jobs.RunDeploy(jobs.DeployConfig{
		DeploymentID:  deployment.ID,
		Image:         req.Image,
		ContainerName: req.ContainerName,
		Port:          req.Port,
		Server: ssh.ServerConfig{
			Host:       req.Server.Host,
			Port:       req.Server.Port,
			User:       req.Server.User,
			PrivateKey: req.Server.PrivateKey,
		},
	})

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf(`{"deployment_id": "%s"}`, deployment.ID)))
}

func main() {
	// signal-aware context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db.Init()
	defer func() {
		fmt.Println("DEFER: closing db...")
		db.Close()
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/docker/ps", dockerPsHandler)
	mux.HandleFunc("/api/deployments", deployHandler)
	mux.HandleFunc("/api/deployments/{id}/logs", handlers.LogsHandler)

	srv := &http.Server{Addr: ":8080", Handler: mux}

	srvErr := make(chan error, 1)
	go func() {
		fmt.Println("Server berjalan di localhost:8080")
		srvErr <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		fmt.Println("\nSignal received, shutting down...")
	case err := <-srvErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println("ListenAndServe error:", err)
			return // to trigger defer
		}
		return
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Println("Shutdown error:", err)
	}
}
