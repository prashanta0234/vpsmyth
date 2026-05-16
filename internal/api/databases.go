package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/prashanta0234/vpsmyth/internal/db"
)

// ── Engine definitions ────────────────────────────────────────────────────────

type engineDef struct {
	Image         string // actual Docker Hub image name
	Versions      []string
	ContainerPort int
	DataDir       string
	DefaultPort   int
	envVars       func(pass string) []string
	extraArgs     func(pass string) []string
	connString    func(pass, bindIP string, port int) string
}

var engines = map[string]engineDef{
	"mysql": {
		Image: "mysql", Versions: []string{"8", "5.7"}, ContainerPort: 3306, DefaultPort: 3306,
		DataDir: "/var/lib/mysql",
		envVars: func(pass string) []string {
			return []string{"-e", "MYSQL_ROOT_PASSWORD=" + pass}
		},
		connString: func(pass, ip string, port int) string {
			return fmt.Sprintf("mysql://root:%s@%s:%d/", pass, ip, port)
		},
	},
	"postgres": {
		Image: "postgres", Versions: []string{"16", "15", "14"}, ContainerPort: 5432, DefaultPort: 5432,
		DataDir: "/var/lib/postgresql/data",
		envVars: func(pass string) []string {
			return []string{"-e", "POSTGRES_PASSWORD=" + pass}
		},
		connString: func(pass, ip string, port int) string {
			return fmt.Sprintf("postgresql://postgres:%s@%s:%d/postgres", pass, ip, port)
		},
	},
	"mongodb": {
		Image: "mongo", Versions: []string{"7", "6"}, ContainerPort: 27017, DefaultPort: 27017,
		DataDir: "/data/db",
		envVars: func(pass string) []string {
			return []string{
				"-e", "MONGO_INITDB_ROOT_USERNAME=admin",
				"-e", "MONGO_INITDB_ROOT_PASSWORD=" + pass,
			}
		},
		connString: func(pass, ip string, port int) string {
			return fmt.Sprintf("mongodb://admin:%s@%s:%d/?authSource=admin", pass, ip, port)
		},
	},
	"redis": {
		Image: "redis", Versions: []string{"7-alpine", "7"}, ContainerPort: 6379, DefaultPort: 6379,
		DataDir: "/data",
		envVars: func(pass string) []string { return nil },
		extraArgs: func(pass string) []string {
			return []string{"redis-server", "--requirepass", pass, "--appendonly", "yes"}
		},
		connString: func(pass, ip string, port int) string {
			return fmt.Sprintf("redis://:%s@%s:%d/0", pass, ip, port)
		},
	},
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func dockerStatus(containerName string) string {
	out, err := exec.Command("docker", "inspect",
		"--format", "{{.State.Status}}", containerName).Output()
	if err != nil {
		return "not found"
	}
	return strings.TrimSpace(string(out))
}

func deployDB(record db.ManagedDB) {
	eng, ok := engines[record.Engine]
	if !ok {
		_ = db.UpdateManagedDBDeployStatus(record.ID, "error", "unknown engine")
		return
	}

	image := eng.Image + ":" + record.Version

	// Pull image
	pullOut, err := exec.Command("docker", "pull", image).CombinedOutput()
	if err != nil {
		_ = db.UpdateManagedDBDeployStatus(record.ID, "error",
			"pull failed: "+strings.TrimSpace(string(pullOut)))
		return
	}

	// Build docker run args
	portBind := fmt.Sprintf("%s:%d:%d", record.BindIP, record.Port, eng.ContainerPort)
	volName := record.ContainerName + "_data"

	args := []string{"run", "-d",
		"--name", record.ContainerName,
		"-p", portBind,
		"-v", volName + ":" + eng.DataDir,
		"--restart", "unless-stopped",
	}
	if env := eng.envVars(record.RootPassword); env != nil {
		args = append(args, env...)
	}
	args = append(args, image)
	if extra := eng.extraArgs; extra != nil {
		args = append(args, extra(record.RootPassword)...)
	}

	runOut, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		_ = db.UpdateManagedDBDeployStatus(record.ID, "error",
			"run failed: "+strings.TrimSpace(string(runOut)))
		return
	}

	_ = db.UpdateManagedDBDeployStatus(record.ID, "ready", "")
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func HandleListDatabases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	dbs, err := db.GetManagedDBs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if dbs == nil {
		dbs = []db.ManagedDB{}
	}
	for i := range dbs {
		if dbs[i].DeployStatus == "ready" {
			dbs[i].DockerStatus = dockerStatus(dbs[i].ContainerName)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dbs)
}

func HandleDeployDatabase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name     string `json:"name"`
		Engine   string `json:"engine"`
		Version  string `json:"version"`
		Password string `json:"password"`
		Port     int    `json:"port"`
		BindIP   string `json:"bind_ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Engine = strings.ToLower(strings.TrimSpace(req.Engine))
	req.BindIP = strings.TrimSpace(req.BindIP)

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		http.Error(w, "password is required", http.StatusBadRequest)
		return
	}
	eng, ok := engines[req.Engine]
	if !ok {
		http.Error(w, "unsupported engine: "+req.Engine, http.StatusBadRequest)
		return
	}
	if req.Port == 0 {
		req.Port = eng.DefaultPort
	}
	if req.BindIP == "" {
		req.BindIP = "127.0.0.1"
	}
	if req.BindIP != "127.0.0.1" && req.BindIP != "0.0.0.0" {
		http.Error(w, "bind_ip must be 127.0.0.1 or 0.0.0.0", http.StatusBadRequest)
		return
	}

	validVersion := false
	for _, v := range eng.Versions {
		if v == req.Version {
			validVersion = true
			break
		}
	}
	if !validVersion {
		req.Version = eng.Versions[0]
	}

	inUse, err := db.IsPortInUse(req.Port)
	if err != nil || inUse {
		http.Error(w, fmt.Sprintf("port %d is already in use by another managed database", req.Port), http.StatusBadRequest)
		return
	}

	containerName := "vpsmyth_db_" + req.Name
	record := db.ManagedDB{
		Name:          req.Name,
		Engine:        req.Engine,
		Version:       req.Version,
		ContainerName: containerName,
		Port:          req.Port,
		BindIP:        req.BindIP,
		RootPassword:  req.Password,
	}

	id, err := db.CreateManagedDB(record)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, "a database with that name already exists", http.StatusConflict)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	record.ID = int(id)

	go deployDB(record)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{"id": id, "status": "deploying"})
}

func HandleDatabaseAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID           int    `json:"id"`
		Action       string `json:"action"`
		DeleteVolume bool   `json:"delete_volume"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID <= 0 {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Action == "delete" && !checkAdminPassword(r) {
		http.Error(w, "Incorrect password", http.StatusUnauthorized)
		return
	}

	record, err := db.GetManagedDB(req.ID)
	if err != nil {
		http.Error(w, "database not found", http.StatusNotFound)
		return
	}

	switch req.Action {
	case "start":
		out, err := exec.Command("docker", "start", record.ContainerName).CombinedOutput()
		if err != nil {
			http.Error(w, strings.TrimSpace(string(out)), http.StatusInternalServerError)
			return
		}
	case "stop":
		out, err := exec.Command("docker", "stop", record.ContainerName).CombinedOutput()
		if err != nil {
			http.Error(w, strings.TrimSpace(string(out)), http.StatusInternalServerError)
			return
		}
	case "restart":
		out, err := exec.Command("docker", "restart", record.ContainerName).CombinedOutput()
		if err != nil {
			http.Error(w, strings.TrimSpace(string(out)), http.StatusInternalServerError)
			return
		}
	case "delete":
		exec.Command("docker", "rm", "-f", record.ContainerName).Run()
		if req.DeleteVolume {
			exec.Command("docker", "volume", "rm", record.ContainerName+"_data").Run()
		}
		if err := db.DeleteManagedDB(req.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "unknown action: "+req.Action, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func HandleDatabaseConnInfo(w http.ResponseWriter, r *http.Request) {
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
	record, err := db.GetManagedDB(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	eng, ok := engines[record.Engine]
	if !ok {
		http.Error(w, "unknown engine", http.StatusInternalServerError)
		return
	}

	connStr := eng.connString(record.RootPassword, record.BindIP, record.Port)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"connection_string": connStr,
		"host":              record.BindIP,
		"port":              record.Port,
		"password":          record.RootPassword,
		"engine":            record.Engine,
	})
}

func HandleGetEngines(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	type engineInfo struct {
		Versions    []string `json:"versions"`
		DefaultPort int      `json:"default_port"`
	}
	result := map[string]engineInfo{}
	for name, eng := range engines {
		result[name] = engineInfo{Versions: eng.Versions, DefaultPort: eng.DefaultPort}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
