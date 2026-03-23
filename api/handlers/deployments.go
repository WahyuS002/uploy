package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/WahyuS002/uploy/auth"
	"github.com/WahyuS002/uploy/broker"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/jobs"
	"github.com/WahyuS002/uploy/respond"
	"github.com/WahyuS002/uploy/ssh"
	"github.com/jackc/pgx/v5"
)

func (s *Server) CreateDeployment(w http.ResponseWriter, r *http.Request) {
	sc, _ := auth.GetSessionContext(r)

	if sc.WorkspaceRole != "owner" && sc.WorkspaceRole != "developer" {
		respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	var req gen.DeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "invalid request body"})
		return
	}

	// Satu query JOIN: application + server + ssh key
	appWithServer, err := db.GetApplicationWithServer(r.Context(), req.ApplicationId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "application not found"})
		} else {
			log.Printf("GetApplicationWithServer id=%s error: %v", req.ApplicationId, err)
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to look up application"})
		}
		return
	}
	if appWithServer.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "application not found"})
		return
	}

	// Load env vars untuk di-inject ke docker run
	envPairs, err := db.GetApplicationEnvPairs(r.Context(), appWithServer.ID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to load environment variables"})
		return
	}

	deployment, err := db.CreateDeployment(context.Background(), sc.WorkspaceID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to create deployment"})
		return
	}

	go jobs.RunDeploy(jobs.DeployConfig{
		DeploymentID:  deployment.ID,
		Image:         appWithServer.Image,
		ContainerName: appWithServer.ContainerName,
		Port:          int(appWithServer.Port),
		EnvVars:       envPairs,
		Server: ssh.ServerConfig{
			Host:       appWithServer.Host,
			Port:       int(appWithServer.ServerPort),
			User:       appWithServer.SSHUser,
			PrivateKey: appWithServer.PrivateKey,
		},
	})

	respond.JSON(w, http.StatusOK, gen.DeployResponse{
		DeploymentId: deployment.ID,
	})
}

func (s *Server) GetDeploymentLogs(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)
	deploymentID := id

	flusher, ok := w.(http.Flusher)
	if !ok {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "streaming not supported"})
		return
	}

	deployment, err := db.GetDeployment(r.Context(), deploymentID)
	if err != nil {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "deployment not found"})
		return
	}

	// Return 404 for cross-workspace access to avoid leaking deployment existence
	if deployment.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "deployment not found"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	sendError := func(msg string) {
		payload, err := json.Marshal(map[string]string{"message": msg})
		if err != nil {
			return
		}
		fmt.Fprintf(w, "event: stream-error\ndata: %s\n\n", payload)
		flusher.Flush()
	}

	sendLog := func(log db.LogEntry) {
		data, err := json.Marshal(log)
		if err != nil {
			return
		}
		fmt.Fprintf(w, "id: %d\ndata: %s\n\n", log.Order, data)
	}

	// Handle reconnect via Last-Event-ID
	var afterOrder int
	if lastID := r.Header.Get("Last-Event-ID"); lastID != "" {
		order, err := strconv.Atoi(lastID)
		if err != nil {
			sendError("invalid Last-Event-ID")
			return
		}
		afterOrder = order
	}

	// 1. Subscribe FIRST so no events are missed during catch-up
	ch := broker.Subscribe(deploymentID)
	defer broker.Unsubscribe(deploymentID, ch)

	// 2. Catch-up from DB
	missed, err := db.GetLogsAfter(r.Context(), deployment.ID, afterOrder)
	if err != nil {
		sendError(err.Error())
		return
	}
	for _, log := range missed {
		sendLog(log)
		afterOrder = log.Order
	}
	if len(missed) > 0 {
		flusher.Flush()
	}

	// 3. Drain broker events that arrived during catch-up (skip duplicates)
drain:
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return // channel closed (slow subscriber)
			}
			if event.Type == broker.Done {
				fmt.Fprintf(w, "event: done\ndata: %s\n\n", event.Status)
				flusher.Flush()
				return
			}
			if event.Order <= afterOrder {
				continue // already sent from DB catch-up
			}
			sendLog(db.LogEntry{ID: event.ID, Order: event.Order, CreatedAt: event.CreatedAt, Output: event.Output, Type: event.LogType})
			afterOrder = event.Order
			flusher.Flush()
		default:
			break drain
		}
	}

	// 4. Re-check deployment status after catch-up
	deployment, err = db.GetDeployment(r.Context(), deploymentID)
	if err != nil {
		sendError(err.Error())
		return
	}
	if deployment.Status == "success" || deployment.Status == "failed" {
		fmt.Fprintf(w, "event: done\ndata: %s\n\n", deployment.Status)
		flusher.Flush()
		return
	}

	// 5. Live stream from broker
	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return // channel closed (slow subscriber)
			}
			switch event.Type {
			case broker.Log:
				sendLog(db.LogEntry{ID: event.ID, Order: event.Order, CreatedAt: event.CreatedAt, Output: event.Output, Type: event.LogType})
				flusher.Flush()
			case broker.Done:
				fmt.Fprintf(w, "event: done\ndata: %s\n\n", event.Status)
				flusher.Flush()
				return
			}
		}
	}
}
