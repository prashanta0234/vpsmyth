package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/prashanta0234/vpsmyth/internal/db"
)

func rangeParams(r string) (since, bucketSecs int64) {
	now := time.Now().Unix()
	switch r {
	case "6h":
		return now - 6*3600, 180
	case "24h":
		return now - 24*3600, 720
	case "7d":
		return now - 7*86400, 5040
	case "30d":
		return now - 30*86400, 21600
	default: // 1h
		return now - 3600, 30
	}
}

func HandleGetSystemMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	since, bucket := rangeParams(r.URL.Query().Get("range"))
	points, err := db.GetSystemMetrics(since, time.Now().Unix(), bucket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if points == nil {
		points = []db.SystemMetricPoint{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": points})
}

func HandleGetContainerMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	containerID := r.URL.Query().Get("id")
	if containerID == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	since, bucket := rangeParams(r.URL.Query().Get("range"))
	points, err := db.GetContainerMetrics(containerID, since, time.Now().Unix(), bucket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if points == nil {
		points = []db.ContainerMetricPoint{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": points})
}

func HandleGetContainersSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	summaries, err := db.GetAllContainersLatest()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if summaries == nil {
		summaries = []db.ContainerSummary{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"containers": summaries})
}

func HandleGetMonitoringApps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	apps, err := db.GetMonitoringApps()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if apps == nil {
		apps = []db.MonitoringApp{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"apps": apps})
}

// HandleGetBreakdown returns system metrics + per-container breakdown in one call,
// using the same time buckets so the data aligns for tooltip rendering on the frontend.
func HandleGetBreakdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	since, bucket := rangeParams(r.URL.Query().Get("range"))
	until := time.Now().Unix()

	system, err := db.GetSystemMetrics(since, until, bucket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if system == nil {
		system = []db.SystemMetricPoint{}
	}

	containers, err := db.GetContainerBreakdown(since, until, bucket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if containers == nil {
		containers = []db.ContainerBreakdownPoint{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"system":     system,
		"containers": containers,
	})
}

func HandleToggleAppMonitoring(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		AppName string `json:"appName"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AppName == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if err := db.SetAppMonitoring(req.AppName, req.Enabled); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}
