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
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/handlers"
	"github.com/WahyuS002/uploy/jobs"
	"github.com/WahyuS002/uploy/respond"
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

	s := &handlers.Server{}

	mux := http.NewServeMux()

	// Spec routes — generated wiring handles routing + path param extraction.
	// Auth middleware checks CookieAuthScopes set by generated wrapper for secured endpoints.
	gen.HandlerWithOptions(s, gen.StdHTTPServerOptions{
		BaseRouter: mux,
		Middlewares: []gen.MiddlewareFunc{
			specAuthMiddleware,
		},
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: err.Error()})
		},
	})

	// OAuth routes — not in spec, manually wired
	mux.HandleFunc("GET /api/auth/github", handlers.GitHubLoginHandler)
	mux.HandleFunc("GET /api/auth/github/callback", handlers.GitHubCallbackHandler)
	mux.HandleFunc("GET /api/auth/google", handlers.GoogleLoginHandler)
	mux.HandleFunc("GET /api/auth/google/callback", handlers.GoogleCallbackHandler)

	go jobs.StartDomainReconciler(ctx)

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

// specAuthMiddleware checks if the generated wrapper marked this endpoint as
// requiring auth (via CookieAuthScopes in context). Public endpoints like
// Register and Login don't have this set, so they pass through directly.
func specAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(gen.CookieAuthScopes) != nil {
			auth.RequireAuth(next).ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
