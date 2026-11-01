package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prashanta0234/vpsmyth/internal/db"
	"github.com/prashanta0234/vpsmyth/internal/system"
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
		// Password lives in a mounted config file (written by deployDB), not in
		// the command line, so `docker inspect` / `ps` never show it.
		extraArgs: func(pass string) []string {
			return []string{"redis-server", "/usr/local/etc/redis/redis.conf"}
		},
		connString: func(pass, ip string, port int) string {
			return fmt.Sprintf("redis://:%s@%s:%d/0", pass, ip, port)
		},
	},
}

// ── Helpers ───────────────────────────────────────────────────────────────────

var dbNameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,40}$`)

// writeRedisConf writes the password into a root-only conf dir (0700) whose
// file (0644) is bind-mounted into the container, readable there by the
// unprivileged redis user but shielded on the host by the directory perms.
func writeRedisConf(containerName, password string) (string, error) {
	dir, err := filepath.Abs(filepath.Join("dbconf", containerName))
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	esc := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(password)
	conf := fmt.Sprintf("requirepass \"%s\"\nappendonly yes\n", esc)
	path := filepath.Join(dir, "redis.conf")
	if err := os.WriteFile(path, []byte(conf), 0644); err != nil {
		return "", err
	}
	return path, nil
}

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

	if record.Engine == "redis" {
		confPath, err := writeRedisConf(record.ContainerName, record.RootPassword)
		if err != nil {
			_ = db.UpdateManagedDBDeployStatus(record.ID, "error", "conf failed: "+err.Error())
			return
		}
		args = append(args, "-v", confPath+":/usr/local/etc/redis/redis.conf:ro")
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

	// Docker's published ports bypass normal ufw rules, so an exposed DB needs
	// its own routed-chain hole (and gets it closed again on delete).
	if record.BindIP == "0.0.0.0" {
		system.UFWRouteAllow(record.Port)
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
	if !dbNameRe.MatchString(req.Name) {
		http.Error(w, "name may only contain letters, numbers, '-' and '_'", http.StatusBadRequest)
		return
	}
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
	if strings.ContainsAny(req.Password, "\n\r\t") {
		http.Error(w, "password must not contain control characters", http.StatusBadRequest)
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
		if record.BindIP == "0.0.0.0" {
			system.UFWRouteDelete(record.Port)
		}
		os.RemoveAll(filepath.Join("dbconf", record.ContainerName))
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

func HandleDatabaseLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	record, err := db.GetManagedDB(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	out, _ := exec.Command("docker", "logs", "--tail", "200", "--timestamps", record.ContainerName).CombinedOutput()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"logs": string(out)})
}

func HandleDatabaseStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	record, err := db.GetManagedDB(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	eng := engines[record.Engine]

	// Started-at and status from docker inspect
	startedAt, _ := exec.Command("docker", "inspect",
		"--format", "{{.State.StartedAt}}", record.ContainerName).Output()
	status, _ := exec.Command("docker", "inspect",
		"--format", "{{.State.Status}}", record.ContainerName).Output()

	// Storage via du inside the container
	storage := "—"
	duOut, err := exec.Command("docker", "exec", record.ContainerName,
		"du", "-sh", eng.DataDir).Output()
	if err == nil {
		parts := strings.Fields(string(duOut))
		if len(parts) > 0 {
			storage = parts[0]
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"started_at": strings.TrimSpace(string(startedAt)),
		"status":     strings.TrimSpace(string(status)),
		"storage":    storage,
	})
}

func HandleDatabaseBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	record, err := db.GetManagedDB(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	eng := engines[record.Engine]
	ts := time.Now().Format("20060102_150405")

	var args []string
	var filename, contentType string

	switch record.Engine {
	case "mysql":
		filename = fmt.Sprintf("%s_%s.sql", record.Name, ts)
		contentType = "application/sql"
		// Password comes from the container's own env so it never appears in `ps`.
		args = []string{"exec", record.ContainerName, "sh", "-c",
			`MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysqldump -uroot --all-databases --no-tablespaces`}
	case "postgres":
		filename = fmt.Sprintf("%s_%s.sql", record.Name, ts)
		contentType = "application/sql"
		args = []string{"exec", record.ContainerName, "sh", "-c",
			`PGPASSWORD="$POSTGRES_PASSWORD" exec pg_dumpall -U postgres`}
	case "mongodb":
		filename = fmt.Sprintf("%s_%s.archive.gz", record.Name, ts)
		contentType = "application/gzip"
		// Password comes from the container's own env so it never appears in `ps`.
		args = []string{"exec", record.ContainerName, "sh", "-c",
			`exec mongodump --username admin --password "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin --archive --gzip`}
	case "redis":
		// Redis has no password env in the container; feed it via stdin so it
		// never appears in `ps` on the host or in the container.
		saveCmd := exec.Command("docker", "exec", "-i", record.ContainerName, "sh", "-c",
			`read -r REDISCLI_AUTH && export REDISCLI_AUTH && exec redis-cli --no-auth-warning SAVE`)
		saveCmd.Stdin = strings.NewReader(record.RootPassword + "\n")
		saveCmd.Run()
		filename = fmt.Sprintf("%s_%s.rdb", record.Name, ts)
		contentType = "application/octet-stream"
		args = []string{"cp", record.ContainerName + ":" + eng.DataDir + "/dump.rdb", "-"}
	default:
		http.Error(w, "unsupported engine", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	cmd := exec.Command("docker", args...)
	cmd.Stdout = w
	cmd.Stderr = io.Discard
	cmd.Run()
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
