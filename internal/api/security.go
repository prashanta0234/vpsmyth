package api

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/prashanta0234/vpsmyth/internal/db"
)

const nginxWhitelistPath = "/etc/nginx/vpsmyth/whitelist.conf"

func HandleGetWhitelist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	entries, err := db.GetWhitelist()
	if err != nil {
		http.Error(w, "Failed to get whitelist", http.StatusInternalServerError)
		return
	}
	if entries == nil {
		entries = []db.WhitelistEntry{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func HandleAddWhitelist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		CIDR  string `json:"cidr"`
		Label string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.CIDR) == "" {
		http.Error(w, "IP or CIDR is required", http.StatusBadRequest)
		return
	}
	normalized, err := db.NormalizeCIDR(req.CIDR)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := db.AddWhitelistEntry(normalized, req.Label); err != nil {
		http.Error(w, "Failed to add entry", http.StatusInternalServerError)
		return
	}
	if err := syncNginxWhitelist(); err != nil {
		// Nginx sync failed — DB is updated, best-effort
		_ = err
	}
	w.WriteHeader(http.StatusCreated)
}

func HandleDeleteWhitelist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	if err := db.DeleteWhitelistEntry(id); err != nil {
		http.Error(w, "Failed to delete entry", http.StatusInternalServerError)
		return
	}
	if err := syncNginxWhitelist(); err != nil {
		_ = err
	}
	w.WriteHeader(http.StatusNoContent)
}

func HandleGetMyIP(w http.ResponseWriter, r *http.Request) {
	ip := realIP(r)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"ip": ip})
}

// realIP extracts the client IP from X-Real-IP (set by Nginx) or falls back to RemoteAddr.
func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// syncNginxWhitelist writes the current whitelist to the Nginx conf file and reloads Nginx.
// An empty whitelist produces an empty file, meaning Nginx allows all IPs.
func syncNginxWhitelist() error {
	entries, err := db.GetWhitelist()
	if err != nil {
		return err
	}

	var content string
	if len(entries) > 0 {
		var sb strings.Builder
		for _, e := range entries {
			sb.WriteString("allow ")
			sb.WriteString(e.CIDR)
			sb.WriteString(";\n")
		}
		// Always allow loopback so Nginx→backend probes work
		sb.WriteString("allow 127.0.0.1;\n")
		sb.WriteString("allow ::1;\n")
		sb.WriteString("deny all;\n")
		content = sb.String()
	}

	if err := os.WriteFile(nginxWhitelistPath, []byte(content), 0644); err != nil {
		return err
	}
	return exec.Command("nginx", "-s", "reload").Run()
}
