package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prashanta0234/vpsmyth/internal/db"
	"github.com/prashanta0234/vpsmyth/internal/stats"
	"github.com/prashanta0234/vpsmyth/internal/system"
)

func HandleStats(w http.ResponseWriter, r *http.Request) {
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

func HandleInstallNode(w http.ResponseWriter, r *http.Request) {
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

func HandleSystemStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := system.GetSystemStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func HandleListContainers(w http.ResponseWriter, r *http.Request) {
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

func HandleContainerAction(action string) http.HandlerFunc {
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

func HandlePullRunContainer(w http.ResponseWriter, r *http.Request) {
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

func HandleContainerLogs(w http.ResponseWriter, r *http.Request) {
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

func HandleInstallTool(name string, installFunc func() error) http.HandlerFunc {
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
