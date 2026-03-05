package main

import (
	"context"
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
	mux.HandleFunc("/api/docker/nginx", dockerNginxHandler)

	srv := &http.Server{Addr: ":8080", Handler: mux}

	srvErr := make(chan error, 1)
	go func() {
		fmt.Println("Server berjalan di localhost:8080")
		srvErr <- srv.ListenAndServe()
	}()

	// tunggu: signal atau server error
	select {
	case <-ctx.Done():
		fmt.Println("\nSignal received, shutting down...")
	case err := <-srvErr:
		// kalau server gagal start / crash
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println("ListenAndServe error:", err)
			return // biar defer jalan
		}
		return
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Println("Shutdown error:", err)
	}

	// main return -> defer db close kepanggil
}
