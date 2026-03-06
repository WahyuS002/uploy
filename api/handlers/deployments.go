package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/WahyuS002/uploy/db"
)

func LogsHandler(w http.ResponseWriter, r *http.Request) {
	deploymentID := r.PathValue("id")

	var after time.Time
	if afterStr := r.URL.Query().Get("after"); afterStr != "" {
		var err error
		after, err = time.Parse(time.RFC3339Nano, afterStr)
		if err != nil {
			http.Error(w, "invalid 'after' parameter (use RFC3339Nano)", 400)
			return
		}
	}

	deployment, err := db.GetDeployment(r.Context(), deploymentID)
	if err != nil {
		http.Error(w, "Deployment not found", 404)
		return
	}

	logs, err := db.GetLogsAfter(deployment.ID, after)
	if err != nil {
		http.Error(w, "failed to fetch logs", 500)
		return
	}
	if logs == nil {
		logs = []db.LogEntry{}
	}

	nextAfter := after.Format(time.RFC3339Nano)
	if len(logs) > 0 {
		nextAfter = logs[len(logs)-1].CreatedAt.Format(time.RFC3339Nano)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": deployment.Status,
		"logs": logs,
		"next_after": nextAfter,
	})
}