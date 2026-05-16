package db

import "time"

type ManagedDB struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Engine        string    `json:"engine"`
	Version       string    `json:"version"`
	ContainerName string    `json:"container_name"`
	Port          int       `json:"port"`
	BindIP        string    `json:"bind_ip"`
	RootPassword  string    `json:"root_password"`
	DeployStatus  string    `json:"deploy_status"`
	DeployError   string    `json:"deploy_error"`
	CreatedAt     time.Time `json:"created_at"`
	// Populated at runtime from Docker, not stored in DB
	DockerStatus string `json:"docker_status,omitempty"`
}

func InitDatabaseTables() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS managed_databases (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			engine TEXT NOT NULL,
			version TEXT NOT NULL,
			container_name TEXT NOT NULL UNIQUE,
			port INTEGER NOT NULL,
			bind_ip TEXT NOT NULL DEFAULT '127.0.0.1',
			root_password TEXT NOT NULL,
			deploy_status TEXT NOT NULL DEFAULT 'deploying',
			deploy_error TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

func GetManagedDBs() ([]ManagedDB, error) {
	rows, err := DB.Query(`
		SELECT id, name, engine, version, container_name, port, bind_ip,
		       root_password, deploy_status, deploy_error, created_at
		FROM managed_databases ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var dbs []ManagedDB
	for rows.Next() {
		var d ManagedDB
		if err := rows.Scan(&d.ID, &d.Name, &d.Engine, &d.Version, &d.ContainerName,
			&d.Port, &d.BindIP, &d.RootPassword, &d.DeployStatus, &d.DeployError, &d.CreatedAt); err != nil {
			return nil, err
		}
		dbs = append(dbs, d)
	}
	return dbs, nil
}

func GetManagedDB(id int) (ManagedDB, error) {
	var d ManagedDB
	err := DB.QueryRow(`
		SELECT id, name, engine, version, container_name, port, bind_ip,
		       root_password, deploy_status, deploy_error, created_at
		FROM managed_databases WHERE id=?`, id).Scan(
		&d.ID, &d.Name, &d.Engine, &d.Version, &d.ContainerName,
		&d.Port, &d.BindIP, &d.RootPassword, &d.DeployStatus, &d.DeployError, &d.CreatedAt,
	)
	return d, err
}

func CreateManagedDB(d ManagedDB) (int64, error) {
	res, err := DB.Exec(`
		INSERT INTO managed_databases (name, engine, version, container_name, port, bind_ip, root_password)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		d.Name, d.Engine, d.Version, d.ContainerName, d.Port, d.BindIP, d.RootPassword,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func UpdateManagedDBDeployStatus(id int, status, errMsg string) error {
	_, err := DB.Exec(
		"UPDATE managed_databases SET deploy_status=?, deploy_error=? WHERE id=?",
		status, errMsg, id,
	)
	return err
}

func DeleteManagedDB(id int) error {
	_, err := DB.Exec("DELETE FROM managed_databases WHERE id=?", id)
	return err
}

func IsPortInUse(port int) (bool, error) {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM managed_databases WHERE port=?", port).Scan(&count)
	return count > 0, err
}
