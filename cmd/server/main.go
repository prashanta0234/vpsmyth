package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/prashanta0234/vpsmyth/internal/db"
	"github.com/prashanta0234/vpsmyth/internal/deploy"
	"github.com/prashanta0234/vpsmyth/internal/stats"
	"github.com/prashanta0234/vpsmyth/internal/system"
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

func main() {
	db.InitDB()
	uiDir := "ui"
	
	http.HandleFunc("/apps/deploy", handleDeploy)
	http.HandleFunc("/apps", handleListApps)
	http.HandleFunc("/apps/stop", handleAppAction("stop"))
	http.HandleFunc("/apps/start", handleAppAction("start"))
	http.HandleFunc("/apps/restart", handleAppAction("restart"))
	http.HandleFunc("/apps/delete", handleAppAction("delete"))
	http.HandleFunc("/apps/update-env", handleUpdateEnv)
	http.HandleFunc("/apps/logs", handleAppLogs)
	http.HandleFunc("/system/install-node", handleInstallNode)
	http.HandleFunc("/system/install-docker", handleInstallTool("Docker", system.InstallDocker))
	http.HandleFunc("/system/install-go", handleInstallTool("Go", system.InstallGo))
	http.HandleFunc("/system/status", handleSystemStatus)
	http.HandleFunc("/system/containers", handleListContainers)
	http.HandleFunc("/system/containers/stop", handleContainerAction("stop"))
	http.HandleFunc("/system/containers/start", handleContainerAction("start"))
	http.HandleFunc("/system/containers/restart", handleContainerAction("restart"))
	http.HandleFunc("/system/containers/delete", handleContainerAction("delete"))
	http.HandleFunc("/system/containers/pull-run", handlePullRunContainer)
	http.HandleFunc("/system/containers/logs", handleContainerLogs)
	http.HandleFunc("/system/settings/dockerhub", handleDockerHubSettings)
	http.HandleFunc("/system/settings/github", handleGitHubSettings)
	http.HandleFunc("/system/settings/secrets", handleSecretsSettings)
	http.HandleFunc("/stats", handleStats)

	// SPA Routing: Serve index.html for unknown routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(uiDir, r.URL.Path)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(uiDir, "index.html"))
			return
		}
		http.FileServer(http.Dir(uiDir)).ServeHTTP(w, r)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("VPSMyth server starting on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleDeploy(w http.ResponseWriter, r *http.Request) {
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

func handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s, err := stats.GetStats()
	if err != nil {
		http.Error(w, "Failed to get stats: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}

func handleInstallNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fmt.Println("Starting Node.js installation on host...")
	err := system.InstallNode()
	if err != nil {
		fmt.Printf("Node.js installation failed: %v\n", err)
		http.Error(w, "Failed to install Node.js: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Node.js installed successfully on host"})
}

func handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := system.GetSystemStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func handleListContainers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	containers, err := system.ListContainers()
	if err != nil {
		http.Error(w, "Failed to list containers: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"containers": containers})
}

func handleContainerAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var err error
		switch action {
		case "stop":
			err = system.StopContainer(req.ID)
		case "start":
			err = system.StartContainer(req.ID)
		case "restart":
			err = system.RestartContainer(req.ID)
		case "delete":
			err = system.DeleteContainer(req.ID)
		}

		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to %s container: %v", action, err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Container %s successfully", action)})
	}
}

func handlePullRunContainer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ImageName     string            `json:"imageName"`
		ContainerName string            `json:"containerName"`
		Port          int               `json:"port"`
		Env           map[string]string `json:"env"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ImageName == "" || req.ContainerName == "" {
		http.Error(w, "Image name and container name are required", http.StatusBadRequest)
		return
	}

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

	err := system.PullAndRunImage(req.ImageName, req.ContainerName, req.Port, req.Env)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Container pulled and started successfully"})
}

func handleContainerLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	logs, err := system.GetContainerLogs(id)
	if err != nil {
		http.Error(w, "Failed to get logs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"logs": logs})
}

func handleDockerHubSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		username, _, err := db.GetDockerHubCredentials()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"username": username})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if err := db.SaveDockerHubCredentials(req.Username, req.Password); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Attempt docker login
		if err := system.LoginDockerHub(req.Username, req.Password); err != nil {
			http.Error(w, "Failed to login to DockerHub: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}
}

func handleGitHubSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		token, err := db.GetGitHubCredentials()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Return masked token or just existence
		hasToken := token != ""
		json.NewEncoder(w).Encode(map[string]interface{}{"hasToken": hasToken})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if err := db.SaveGitHubCredentials(req.Token); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}
}

func handleSecretsSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		secrets, err := db.GetGlobalSecrets()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(secrets)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Action string `json:"action"` // "save" or "delete"
			Key    string `json:"key"`
			Value  string `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		var err error
		if req.Action == "delete" {
			err = db.DeleteSecret(req.Key)
		} else {
			err = db.SaveSecret(req.Key, req.Value)
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}
}

func handleInstallTool(name string, installFunc func() error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		fmt.Printf("Starting %s installation on host...\n", name)
		err := installFunc()
		if err != nil {
			fmt.Printf("%s installation failed: %v\n", name, err)
			http.Error(w, fmt.Sprintf("Failed to install %s: %v", name, err), http.StatusInternalServerError)
			return
		}

		fmt.Printf("%s installed successfully!\n", name)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("%s installed successfully on host", name)})
	}
}

func handleListApps(w http.ResponseWriter, r *http.Request) {
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

func handleAppAction(action string) http.HandlerFunc {
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

func handleUpdateEnv(w http.ResponseWriter, r *http.Request) {
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

func handleAppLogs(w http.ResponseWriter, r *http.Request) {
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
