package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/WahyuS002/uploy/auth"
	"github.com/WahyuS002/uploy/config"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/handlers"
	"github.com/joho/godotenv"
)

func main() {
	// signal-aware context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	_ = godotenv.Load()

	if err := config.Load(); err != nil {
		log.Fatal("Config error: ", err)
	}

	db.Init(config.C.DatabaseURL)
	defer func() {
		fmt.Println("DEFER: closing db...")
		db.Close()
	}()

	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("POST /api/auth/register", handlers.RegisterHandler)
	mux.HandleFunc("POST /api/auth/login", handlers.LoginHandler)
	mux.HandleFunc("GET /api/auth/github", handlers.GitHubLoginHandler)
	mux.HandleFunc("GET /api/auth/github/callback", handlers.GitHubCallbackHandler)
	mux.HandleFunc("GET /api/auth/google", handlers.GoogleLoginHandler)
	mux.HandleFunc("GET /api/auth/google/callback", handlers.GoogleCallbackHandler)

	// Protected routes
	mux.Handle("POST /api/auth/logout", auth.RequireAuth(http.HandlerFunc(handlers.LogoutHandler)))
	mux.Handle("GET /api/auth/me", auth.RequireAuth(http.HandlerFunc(handlers.MeHandler)))
	mux.Handle("POST /api/deployments", auth.RequireAuth(auth.RequireRole("owner", "developer")(http.HandlerFunc(handlers.DeployHandler))))
	mux.Handle("GET /api/deployments/{id}/logs", auth.RequireAuth(http.HandlerFunc(handlers.LogsHandler)))

	srv := &http.Server{Addr: ":8080", Handler: mux}

	srvErr := make(chan error, 1)
	go func() {
		fmt.Println("Server running on localhost:8080")
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
