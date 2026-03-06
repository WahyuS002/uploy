package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
	w.Header().Set("Connection", "keep-alive")

	var after time.Time
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			logs, err := db.GetLogsAfter(deployment.ID, after)
			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
				flusher.Flush()
				return
			}

			for _, log := range logs {
				data, _ := json.Marshal(log)
				fmt.Fprintf(w, "data: %s\n\n", data)
				after = log.CreatedAt
			}

			deployment, err = db.GetDeployment(r.Context(), deploymentID)
			if err != nil {
				return
			}

			if deployment.Status == "success" || deployment.Status == "failed" {
				fmt.Fprintf(w, "event: done\ndata: %s\n\n", deployment.Status)
				flusher.Flush()
				return
			}

			flusher.Flush()
		}
	}
}