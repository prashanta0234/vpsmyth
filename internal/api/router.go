package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/prashanta0234/vpsmyth/internal/auth"
	"github.com/prashanta0234/vpsmyth/internal/system"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Public routes
		if r.URL.Path == "/login.html" || r.URL.Path == "/api/auth/login" || strings.HasPrefix(r.URL.Path, "/css/") || strings.HasPrefix(r.URL.Path, "/js/") || strings.HasPrefix(r.URL.Path, "/assets/") {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie("vpsmyth_token")
		if err != nil {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			} else {
				http.Redirect(w, r, "/login.html", http.StatusSeeOther)
			}
			return
		}

		_, err = auth.ValidateToken(cookie.Value)
		if err != nil {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			} else {
				http.Redirect(w, r, "/login.html", http.StatusSeeOther)
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}

func RegisterRoutes(mux *http.ServeMux) {
	// Auth routes
	mux.HandleFunc("/api/auth/login", HandleLogin)
	mux.HandleFunc("/api/auth/logout", HandleLogout)

	// App routes
	mux.HandleFunc("/api/apps/deploy", HandleDeploy)
	mux.HandleFunc("/api/apps", HandleListApps)
	mux.HandleFunc("/api/apps/stop", HandleAppAction("stop"))
	mux.HandleFunc("/api/apps/start", HandleAppAction("start"))
	mux.HandleFunc("/api/apps/restart", HandleAppAction("restart"))
	mux.HandleFunc("/api/apps/delete", HandleAppAction("delete"))
	mux.HandleFunc("/api/apps/update-env", HandleUpdateEnv)
	mux.HandleFunc("/api/apps/logs", HandleAppLogs)

	// System routes
	mux.HandleFunc("/api/system/install-node", HandleInstallNode)
	mux.HandleFunc("/api/system/install-docker", HandleInstallTool("Docker", system.InstallDocker))
	mux.HandleFunc("/api/system/install-go", HandleInstallTool("Go", system.InstallGo))
	mux.HandleFunc("/api/system/status", HandleSystemStatus)
	mux.HandleFunc("/api/system/containers", HandleListContainers)
	mux.HandleFunc("/api/system/containers/stop", HandleContainerAction("stop"))
	mux.HandleFunc("/api/system/containers/start", HandleContainerAction("start"))
	mux.HandleFunc("/api/system/containers/restart", HandleContainerAction("restart"))
	mux.HandleFunc("/api/system/containers/delete", HandleContainerAction("delete"))
	mux.HandleFunc("/api/system/containers/pull-run", HandlePullRunContainer)
	mux.HandleFunc("/api/system/containers/logs", HandleContainerLogs)

	// Settings routes
	mux.HandleFunc("/api/system/settings/dockerhub", HandleDockerHubSettings)
	mux.HandleFunc("/api/system/settings/github", HandleGitHubSettings)
	mux.HandleFunc("/api/system/settings/secrets", HandleSecretsSettings)

	// Stats route
	mux.HandleFunc("/api/stats", HandleStats)

	// SPA Routing
	uiDir := "ui"
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(uiDir, r.URL.Path)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(uiDir, "index.html"))
			return
		}
		http.FileServer(http.Dir(uiDir)).ServeHTTP(w, r)
	})
}
