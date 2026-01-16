package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prashanta0234/vpsmyth/internal/db"
	"github.com/prashanta0234/vpsmyth/internal/deploy"
)

// DeployRequest represents the expected JSON body for the /apps/deploy endpoint.
type DeployRequest struct {
	AppName    string            `json:"appName"`
	DeployType string            `json:"deployType"` // "git" or "image"
	Category   string            `json:"category"`
	Framework  string            `json:"framework"`
	RepoURL    string            `json:"repoURL"`
	ImageName  string            `json:"imageName"`
	Port       int               `json:"port"`
	Env        map[string]string `json:"env"`
}

func HandleDeploy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AppName == "" || req.RepoURL == "" || req.Port == 0 {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	fmt.Printf("Received deployment request for: %s\n", req.AppName)

	// Inject global secrets
	globalSecrets, _ := db.GetGlobalSecrets()
	if req.Env == nil {
		req.Env = make(map[string]string)
	}
	for k, v := range globalSecrets {
		if _, exists := req.Env[k]; !exists {
			req.Env[k] = v
		}
	}

	var err error
	if req.DeployType == "image" {
		err = deploy.DeployFromImage(req.AppName, req.ImageName, req.Port, req.Env)
	} else {
		err = deploy.DeployNodeDocker(req.AppName, req.Category, req.Framework, req.RepoURL, req.Port, req.Env)
	}

	if err != nil {
		fmt.Printf("Deployment failed: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Deployment started successfully", "appName": req.AppName})
}

func HandleListApps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apps, err := deploy.ListApps()
	if err != nil {
		http.Error(w, "Failed to list apps: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"apps": apps})
}

func HandleAppAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			AppName string `json:"appName"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var err error
		switch action {
		case "stop":
			err = deploy.StopApp(req.AppName)
		case "start":
			err = deploy.StartApp(req.AppName)
		case "restart":
			err = deploy.RestartApp(req.AppName)
		case "delete":
			err = deploy.DeleteApp(req.AppName)
		}

		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to %s app: %v", action, err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("App %s successfully", action)})
	}
}

func HandleUpdateEnv(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AppName string            `json:"appName"`
		Env     map[string]string `json:"env"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := deploy.UpdateAppEnv(req.AppName, req.Env)
	if err != nil {
		http.Error(w, "Failed to update environment variables: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Environment variables updated and app restarted successfully"})
}

func HandleAppLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	appName := r.URL.Query().Get("appName")
	if appName == "" {
		http.Error(w, "Missing appName parameter", http.StatusBadRequest)
		return
	}

	logs, err := deploy.GetLogs(appName)
	if err != nil {
		http.Error(w, "Failed to get logs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"logs": logs})
}
