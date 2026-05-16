package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/prashanta0234/vpsmyth/internal/auth"
	"github.com/prashanta0234/vpsmyth/internal/db"
	"github.com/prashanta0234/vpsmyth/internal/system"
)

// IPWhitelistMiddleware blocks requests whose X-Real-IP is not in the whitelist.
// When the whitelist is empty, all IPs are allowed. Direct local connections
// (no X-Real-IP header) pass through because the backend is bound to 127.0.0.1.
func IPWhitelistMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.Header.Get("X-Real-IP")
		if ip == "" {
			next.ServeHTTP(w, r)
			return
		}
		allowed, err := db.IsIPAllowed(strings.TrimSpace(ip))
		if err != nil || !allowed {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loginURL() string {
	if p := os.Getenv("PANEL_PATH"); p != "" {
		return "/" + p + "/login.html"
	}
	return "/login.html"
}

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
				http.Redirect(w, r, loginURL(), http.StatusSeeOther)
			}
			return
		}

		_, err = auth.ValidateToken(cookie.Value)
		if err != nil {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			} else {
				http.Redirect(w, r, loginURL(), http.StatusSeeOther)
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

	// Firewall routes
	mux.HandleFunc("/api/firewall/status", HandleGetFirewallStatus)
	mux.HandleFunc("/api/firewall/enable", HandleFirewallEnable)
	mux.HandleFunc("/api/firewall/disable", HandleFirewallDisable)
	mux.HandleFunc("/api/firewall/rule/add", HandleAddFirewallRule)
	mux.HandleFunc("/api/firewall/rule/delete", HandleDeleteFirewallRule)

	// Security routes
	mux.HandleFunc("/api/security/whitelist", HandleGetWhitelist)
	mux.HandleFunc("/api/security/whitelist/add", HandleAddWhitelist)
	mux.HandleFunc("/api/security/whitelist/delete", HandleDeleteWhitelist)
	mux.HandleFunc("/api/security/myip", HandleGetMyIP)

	// Database routes
	mux.HandleFunc("/api/databases", HandleListDatabases)
	mux.HandleFunc("/api/databases/engines", HandleGetEngines)
	mux.HandleFunc("/api/databases/deploy", HandleDeployDatabase)
	mux.HandleFunc("/api/databases/action", HandleDatabaseAction)
	mux.HandleFunc("/api/databases/conninfo", HandleDatabaseConnInfo)

	// Cron routes
	mux.HandleFunc("/api/cron/jobs", HandleListCronJobs)
	mux.HandleFunc("/api/cron/jobs/add", HandleAddCronJob)
	mux.HandleFunc("/api/cron/jobs/update", HandleUpdateCronJob)
	mux.HandleFunc("/api/cron/jobs/delete", HandleDeleteCronJob)
	mux.HandleFunc("/api/cron/jobs/toggle", HandleToggleCronJob)
	mux.HandleFunc("/api/cron/jobs/run", HandleRunCronJob)
	mux.HandleFunc("/api/cron/jobs/runs", HandleGetCronRuns)

	// Monitoring routes
	mux.HandleFunc("/api/monitoring/system", HandleGetSystemMetrics)
	mux.HandleFunc("/api/monitoring/breakdown", HandleGetBreakdown)
	mux.HandleFunc("/api/monitoring/containers", HandleGetContainersSummary)
	mux.HandleFunc("/api/monitoring/container", HandleGetContainerMetrics)
	mux.HandleFunc("/api/monitoring/apps", HandleGetMonitoringApps)
	mux.HandleFunc("/api/monitoring/app/toggle", HandleToggleAppMonitoring)

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
