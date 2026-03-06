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

	// Handle reconnect via Last-Event-ID
	var afterID int64
	if lastID := r.Header.Get("Last-Event-ID"); lastID != "" {
		afterID, _ = strconv.ParseInt(lastID, 10, 64)
	}

	// Send missed logs from DB
	missed, err := db.GetLogsAfter(r.Context(), deployment.ID, afterID)
	if err != nil {
		sendError(err.Error())
		return
	}
	for _, log := range missed {
		data, err := json.Marshal(log)
		if err != nil {
			continue
		}
		fmt.Fprintf(w, "id: %d\ndata: %s\n\n", log.ID, data)
		afterID = log.ID
	}
	if len(missed) > 0 {
		flusher.Flush()
	}

	// Already done? send final event and return
	if deployment.Status == "success" || deployment.Status == "failed" {
		fmt.Fprintf(w, "event: done\ndata: %s\n\n", deployment.Status)
		flusher.Flush()
		return
	}

	// Subscribe to broker for real-time events
	ch := broker.Subscribe(deploymentID)
	defer broker.Unsubscribe(deploymentID, ch)

	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-ch:
			switch event.Type {
			case broker.Log:
				data, err := json.Marshal(db.LogEntry{
					ID:        event.ID,
					CreatedAt: event.CreatedAt,
					Output:    event.Output,
				})
				if err != nil {
					continue
				}
				fmt.Fprintf(w, "id: %d\ndata: %s\n\n", event.ID, data)
				flusher.Flush()
			case broker.Done:
				fmt.Fprintf(w, "event: done\ndata: %s\n\n", event.Status)
				flusher.Flush()
				return
			}
		}
	}
}
