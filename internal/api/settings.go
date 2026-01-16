package api

import (
	"encoding/json"
	"net/http"

	"github.com/prashanta0234/vpsmyth/internal/db"
	"github.com/prashanta0234/vpsmyth/internal/system"
)

func HandleDockerHubSettings(w http.ResponseWriter, r *http.Request) {
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

func HandleGitHubSettings(w http.ResponseWriter, r *http.Request) {
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

func HandleSecretsSettings(w http.ResponseWriter, r *http.Request) {
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
