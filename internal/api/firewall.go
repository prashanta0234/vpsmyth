package api

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/prashanta0234/vpsmyth/internal/auth"
	"github.com/prashanta0234/vpsmyth/internal/db"
)

type FirewallRule struct {
	Number int    `json:"number"`
	To     string `json:"to"`
	Action string `json:"action"`
	From   string `json:"from"`
	IsIPv6 bool   `json:"is_ipv6"`
}

type FirewallStatus struct {
	Enabled bool           `json:"enabled"`
	Rules   []FirewallRule `json:"rules"`
}

var ufwRuleRe = regexp.MustCompile(
	`^\[\s*(\d+)\]\s+(.+?)\s{2,}(ALLOW IN|ALLOW OUT|ALLOW FWD|DENY IN|DENY OUT|DENY FWD|LIMIT IN|LIMIT OUT|REJECT IN|REJECT OUT|REJECT FWD|ALLOW|DENY|LIMIT|REJECT)\s+(.+?)\s*$`,
)

// checkAdminPassword verifies the X-Admin-Password header against the stored
// admin password hash. Returns false if the header is missing or wrong.
func checkAdminPassword(r *http.Request) bool {
	pw := r.Header.Get("X-Admin-Password")
	if pw == "" {
		return false
	}
	var hash string
	if err := db.DB.QueryRow("SELECT password_hash FROM users LIMIT 1").Scan(&hash); err != nil {
		return false
	}
	ok, err := auth.VerifyPassword(pw, hash)
	return err == nil && ok
}

// runUFW executes ufw directly (works when the process is root) and falls
// back to sudo if the process lacks root privileges (requires the NOPASSWD
// sudoers entry added by install.sh).
func runUFW(args ...string) (string, error) {
	out, err := exec.Command("ufw", args...).CombinedOutput()
	outStr := strings.TrimSpace(string(out))
	if err == nil {
		return outStr, nil
	}
	lower := strings.ToLower(outStr)
	if strings.Contains(lower, "you need to be root") || strings.Contains(lower, "permission denied") {
		out, err = exec.Command("sudo", append([]string{"ufw"}, args...)...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	return outStr, err
}

func parseUFWOutput(raw string) FirewallStatus {
	var status FirewallStatus
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Status: active") {
			status.Enabled = true
			continue
		}
		m := ufwRuleRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		num, _ := strconv.Atoi(strings.TrimSpace(m[1]))
		to := strings.TrimSpace(m[2])
		action := strings.TrimSpace(m[3])
		from := strings.TrimSpace(m[4])

		isIPv6 := strings.Contains(to, "(v6)") || strings.Contains(from, "(v6)")
		to = strings.ReplaceAll(to, " (v6)", "")
		from = strings.ReplaceAll(from, " (v6)", "")

		status.Rules = append(status.Rules, FirewallRule{
			Number: num, To: to, Action: action, From: from, IsIPv6: isIPv6,
		})
	}
	return status
}

func HandleGetFirewallStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if !checkAdminPassword(r) {
		http.Error(w, "Incorrect password", http.StatusUnauthorized)
		return
	}
	out, err := runUFW("status", "numbered")
	if err != nil {
		http.Error(w, "ufw error: "+out, http.StatusInternalServerError)
		return
	}
	status := parseUFWOutput(out)
	if status.Rules == nil {
		status.Rules = []FirewallRule{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func HandleFirewallEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if !checkAdminPassword(r) {
		http.Error(w, "Incorrect password", http.StatusUnauthorized)
		return
	}
	out, err := runUFW("--force", "enable")
	if err != nil {
		http.Error(w, out, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func HandleFirewallDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if !checkAdminPassword(r) {
		http.Error(w, "Incorrect password", http.StatusUnauthorized)
		return
	}
	out, err := runUFW("--force", "disable")
	if err != nil {
		http.Error(w, out, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type AddRuleReq struct {
	Action    string `json:"action"`
	Direction string `json:"direction"`
	Port      string `json:"port"`
	Proto     string `json:"proto"`
	FromIP    string `json:"from_ip"`
}

func HandleAddFirewallRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if !checkAdminPassword(r) {
		http.Error(w, "Incorrect password", http.StatusUnauthorized)
		return
	}

	var req AddRuleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	req.Action = strings.ToLower(strings.TrimSpace(req.Action))
	req.Port = strings.TrimSpace(req.Port)
	req.Proto = strings.ToLower(strings.TrimSpace(req.Proto))
	req.Direction = strings.ToLower(strings.TrimSpace(req.Direction))
	req.FromIP = strings.TrimSpace(req.FromIP)

	if req.Action != "allow" && req.Action != "deny" && req.Action != "limit" {
		http.Error(w, "action must be allow, deny, or limit", http.StatusBadRequest)
		return
	}
	if req.Port == "" {
		http.Error(w, "port is required", http.StatusBadRequest)
		return
	}
	if req.Proto != "tcp" && req.Proto != "udp" && req.Proto != "any" {
		http.Error(w, "proto must be tcp, udp, or any", http.StatusBadRequest)
		return
	}

	args := []string{req.Action}
	if req.Direction == "in" || req.Direction == "out" {
		args = append(args, req.Direction)
	}
	if req.FromIP != "" {
		args = append(args, "from", req.FromIP, "to", "any", "port", req.Port)
		if req.Proto != "any" {
			args = append(args, "proto", req.Proto)
		}
	} else {
		portSpec := req.Port
		if req.Proto != "any" {
			portSpec += "/" + req.Proto
		}
		args = append(args, portSpec)
	}

	out, err := runUFW(args...)
	if err != nil {
		http.Error(w, out, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func HandleDeleteFirewallRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if !checkAdminPassword(r) {
		http.Error(w, "Incorrect password", http.StatusUnauthorized)
		return
	}

	var req struct {
		Number int `json:"number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Number <= 0 {
		http.Error(w, "invalid rule number", http.StatusBadRequest)
		return
	}

	out, err := runUFW("--force", "delete", strconv.Itoa(req.Number))
	if err != nil {
		http.Error(w, out, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
