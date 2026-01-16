package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

// InitDB initializes the SQLite database and creates necessary tables.
func InitDB(dbPath string) error {
	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}

	createTables := `
	CREATE TABLE IF NOT EXISTS credentials (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		service TEXT UNIQUE,
		username TEXT,
		password TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS secrets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT UNIQUE,
		value TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err = DB.Exec(createTables)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	fmt.Println("Database initialized successfully.")
	return nil
}

// GetDockerHubCredentials retrieves DockerHub username and password.
func GetDockerHubCredentials() (string, string, error) {
	var username, password string
	err := DB.QueryRow("SELECT username, password FROM credentials WHERE service = 'dockerhub'").Scan(&username, &password)
	if err == sql.ErrNoRows {
		return "", "", nil
	}
	return username, password, err
}

// SaveDockerHubCredentials saves DockerHub username and password.
func SaveDockerHubCredentials(username, password string) error {
	_, err := DB.Exec("INSERT OR REPLACE INTO credentials (service, username, password, updated_at) VALUES ('dockerhub', ?, ?, CURRENT_TIMESTAMP)", username, password)
	return err
}

// GetGitHubCredentials retrieves GitHub token.
func GetGitHubCredentials() (string, error) {
	var token string
	err := DB.QueryRow("SELECT password FROM credentials WHERE service = 'github'").Scan(&token)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return token, err
}

// SaveGitHubCredentials saves GitHub token.
func SaveGitHubCredentials(token string) error {
	_, err := DB.Exec("INSERT OR REPLACE INTO credentials (service, username, password, updated_at) VALUES ('github', 'github_token', ?, CURRENT_TIMESTAMP)", token)
	return err
}

// GetGlobalSecrets retrieves all global secrets as a map.
func GetGlobalSecrets() (map[string]string, error) {
	rows, err := DB.Query("SELECT key, value FROM secrets")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	secrets := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		secrets[key] = value
	}
	return secrets, nil
}

// SaveSecret saves or updates a global secret.
func SaveSecret(key, value string) error {
	_, err := DB.Exec("INSERT OR REPLACE INTO secrets (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)", key, value)
	return err
}

// DeleteSecret removes a global secret.
func DeleteSecret(key string) error {
	_, err := DB.Exec("DELETE FROM secrets WHERE key = ?", key)
	return err
}
