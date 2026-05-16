package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	cronrunner "github.com/prashanta0234/vpsmyth/internal/cron"
	"github.com/prashanta0234/vpsmyth/internal/db"
)

func HandleListCronJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	jobs, err := db.GetCronJobs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if jobs == nil {
		jobs = []db.CronJob{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

func HandleAddCronJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var job db.CronJob
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if msg := validateJob(job); msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	job.Enabled = true
	id, err := db.CreateCronJob(job)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

func HandleUpdateCronJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var job db.CronJob
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if job.ID <= 0 {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	if msg := validateJob(job); msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	if err := db.UpdateCronJob(job); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func HandleDeleteCronJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := db.DeleteCronJob(req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func HandleToggleCronJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID      int  `json:"id"`
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID <= 0 {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := db.ToggleCronJob(req.ID, req.Enabled); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func HandleRunCronJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := cronrunner.RunNow(req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func HandleGetCronRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	runs, err := db.GetCronRuns(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if runs == nil {
		runs = []db.CronRun{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runs)
}

func validateJob(job db.CronJob) string {
	if strings.TrimSpace(job.Name) == "" {
		return "name is required"
	}
	if msg := cronrunner.ValidateSchedule(job.Schedule); msg != "" {
		return "invalid schedule: " + msg
	}
	switch job.Type {
	case "command":
		if strings.TrimSpace(job.Command) == "" {
			return "command is required"
		}
	case "script":
		if strings.TrimSpace(job.ScriptContent) == "" {
			return "script content is required"
		}
	case "curl":
		if strings.TrimSpace(job.CurlURL) == "" {
			return "URL is required"
		}
	default:
		return "type must be command, script, or curl"
	}
	return ""
}
