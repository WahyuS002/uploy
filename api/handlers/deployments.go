package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/WahyuS002/uploy/broker"
	"github.com/WahyuS002/uploy/db"
)

func LogsHandler(w http.ResponseWriter, r *http.Request) {
	deploymentID := r.PathValue("id")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", 500)
		return
	}

	deployment, err := db.GetDeployment(r.Context(), deploymentID)
	if err != nil {
		http.Error(w, "Deployment not found", 404)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	sendError := func(msg string) {
		payload, _ := json.Marshal(map[string]string{"message": msg})
		fmt.Fprintf(w, "event: stream-error\ndata: %s\n\n", payload)
		flusher.Flush()
	}

	sendLog := func(log db.LogEntry) {
		data, err := json.Marshal(log)
		if err != nil {
			return
		}
		fmt.Fprintf(w, "id: %d\ndata: %s\n\n", log.ID, data)
	}

	// Handle reconnect via Last-Event-ID
	var afterID int64
	if lastID := r.Header.Get("Last-Event-ID"); lastID != "" {
		id, err := strconv.ParseInt(lastID, 10, 64)
		if err != nil {
			sendError("invalid Last-Event-ID")
			return
		}
		afterID = id
	}

	// 1. Subscribe FIRST so no events are missed during catch-up
	ch := broker.Subscribe(deploymentID)
	defer broker.Unsubscribe(deploymentID, ch)

	// 2. Catch-up from DB
	missed, err := db.GetLogsAfter(r.Context(), deployment.ID, afterID)
	if err != nil {
		sendError(err.Error())
		return
	}
	for _, log := range missed {
		sendLog(log)
		afterID = log.ID
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
			if event.ID <= afterID {
				continue // already sent from DB catch-up
			}
			sendLog(db.LogEntry{ID: event.ID, CreatedAt: event.CreatedAt, Output: event.Output})
			afterID = event.ID
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
				sendLog(db.LogEntry{ID: event.ID, CreatedAt: event.CreatedAt, Output: event.Output})
				flusher.Flush()
			case broker.Done:
				fmt.Fprintf(w, "event: done\ndata: %s\n\n", event.Status)
				flusher.Flush()
				return
			}
		}
	}
}
