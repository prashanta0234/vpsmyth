package api

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/prashanta0234/vpsmyth/internal/system"
)

func RegisterRoutes(mux *http.ServeMux) {
	// App routes
	mux.HandleFunc("/apps/deploy", HandleDeploy)
	mux.HandleFunc("/apps", HandleListApps)
	mux.HandleFunc("/apps/stop", HandleAppAction("stop"))
	mux.HandleFunc("/apps/start", HandleAppAction("start"))
	mux.HandleFunc("/apps/restart", HandleAppAction("restart"))
	mux.HandleFunc("/apps/delete", HandleAppAction("delete"))
	mux.HandleFunc("/apps/update-env", HandleUpdateEnv)
	mux.HandleFunc("/apps/logs", HandleAppLogs)

	// System routes
	mux.HandleFunc("/system/install-node", HandleInstallNode)
	mux.HandleFunc("/system/install-docker", HandleInstallTool("Docker", system.InstallDocker))
	mux.HandleFunc("/system/install-go", HandleInstallTool("Go", system.InstallGo))
	mux.HandleFunc("/system/status", HandleSystemStatus)
	mux.HandleFunc("/system/containers", HandleListContainers)
	mux.HandleFunc("/system/containers/stop", HandleContainerAction("stop"))
	mux.HandleFunc("/system/containers/start", HandleContainerAction("start"))
	mux.HandleFunc("/system/containers/restart", HandleContainerAction("restart"))
	mux.HandleFunc("/system/containers/delete", HandleContainerAction("delete"))
	mux.HandleFunc("/system/containers/pull-run", HandlePullRunContainer)
	mux.HandleFunc("/system/containers/logs", HandleContainerLogs)

	// Settings routes
	mux.HandleFunc("/system/settings/dockerhub", HandleDockerHubSettings)
	mux.HandleFunc("/system/settings/github", HandleGitHubSettings)
	mux.HandleFunc("/system/settings/secrets", HandleSecretsSettings)

	// Stats route
	mux.HandleFunc("/stats", HandleStats)

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
