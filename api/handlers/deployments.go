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

	svcWithServer, err := db.GetServiceWithServer(r.Context(), req.ServiceId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "service not found"})
		} else {
			log.Printf("GetServiceWithServer id=%s error: %v", req.ServiceId, err)
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to look up service"})
		}
		return
	}
	if svcWithServer.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "service not found"})
		return
	}

	if svcWithServer.Kind != "application" {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "only application services can be deployed"})
		return
	}

	// Load env vars and domains
	envPairs, err := db.GetServiceEnvPairs(r.Context(), svcWithServer.ID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to load environment variables"})
		return
	}

	svcDomains, err := db.ListDomainsByService(r.Context(), svcWithServer.ID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to load domains"})
		return
	}
	domainNames := make([]string, len(svcDomains))
	for i, d := range svcDomains {
		domainNames[i] = d.Domain
	}

	deployment, err := db.CreateDeployment(context.Background(), sc.WorkspaceID, svcWithServer.ID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to create deployment"})
		return
	}

	go jobs.RunDeploy(jobs.DeployConfig{
		DeploymentID:  deployment.ID,
		ServiceID:     svcWithServer.ID,
		Image:         svcWithServer.Image,
		ContainerName: svcWithServer.ContainerName,
		Port:          int(svcWithServer.Port),
		EnvVars:       envPairs,
		Domains:       domainNames,
		ServerID:      svcWithServer.ServerID,
		Server: ssh.ServerConfig{
			Host:       svcWithServer.Host,
			Port:       int(svcWithServer.ServerPort),
			User:       svcWithServer.SSHUser,
			PrivateKey: svcWithServer.PrivateKey,
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
			sendLog(db.LogEntry{ID: event.ID, Order: event.Order, CreatedAt: event.CreatedAt, Output: event.Output, Type: event.LogType, Phase: event.Phase})
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
				sendLog(db.LogEntry{ID: event.ID, Order: event.Order, CreatedAt: event.CreatedAt, Output: event.Output, Type: event.LogType, Phase: event.Phase})
				flusher.Flush()
			case broker.Done:
				fmt.Fprintf(w, "event: done\ndata: %s\n\n", event.Status)
				flusher.Flush()
				return
			}
		}
	}
}

func (s *Server) ListServiceDeployments(w http.ResponseWriter, r *http.Request, id string, params gen.ListServiceDeploymentsParams) {
	sc, _ := auth.GetSessionContext(r)

	svc, err := db.GetServiceByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "service not found"})
		} else {
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to get service"})
		}
		return
	}
	if svc.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "service not found"})
		return
	}

	limit := int32(20)
	if params.Limit != nil && *params.Limit > 0 {
		limit = int32(*params.Limit)
	}

	deployments, err := db.ListDeploymentsByService(r.Context(), svc.ID, limit)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to list deployments"})
		return
	}

	resp := make([]gen.DeploymentResponse, len(deployments))
	for i, d := range deployments {
		resp[i] = gen.DeploymentResponse{
			Id:        d.ID,
			Status:    gen.DeploymentResponseStatus(d.Status),
			ServiceId: d.ServiceID,
			CreatedAt: d.CreatedAt,
		}
	}

	respond.JSON(w, http.StatusOK, resp)
}
