package db

import (
	"database/sql"
	"fmt"
	"time"

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
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE,
		password_hash TEXT,
		failed_attempts INTEGER DEFAULT 0,
		locked_until DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
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

// User represents a system user.
type User struct {
	ID             int
	Username       string
	PasswordHash   string
	FailedAttempts int
	LockedUntil    *time.Time
}

// CreateUser adds a new user to the database.
func CreateUser(username, passwordHash string) error {
	_, err := DB.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", username, passwordHash)
	return err
}

// GetUserByUsername retrieves a user by their username.
func GetUserByUsername(username string) (User, error) {
	var user User
	var lockedUntil sql.NullTime
	err := DB.QueryRow("SELECT id, username, password_hash, failed_attempts, locked_until FROM users WHERE username = ?", username).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.FailedAttempts, &lockedUntil)
	if lockedUntil.Valid {
		user.LockedUntil = &lockedUntil.Time
	}
	return user, err
}

// IncrementFailedAttempts increases the failed login count and locks the account if threshold reached.
func IncrementFailedAttempts(username string, maxAttempts int, lockoutDuration time.Duration) error {
	var attempts int
	err := DB.QueryRow("SELECT failed_attempts FROM users WHERE username = ?", username).Scan(&attempts)
	if err != nil {
		return err
	}

	attempts++
	var lockedUntil interface{} = nil
	if attempts >= maxAttempts {
		lockedUntil = time.Now().Add(lockoutDuration)
	}

	_, err = DB.Exec("UPDATE users SET failed_attempts = ?, locked_until = ? WHERE username = ?", attempts, lockedUntil, username)
	return err
}

// ResetFailedAttempts resets the failed login count and unlocks the account.
func ResetFailedAttempts(username string) error {
	_, err := DB.Exec("UPDATE users SET failed_attempts = 0, locked_until = NULL WHERE username = ?", username)
	return err
}

// HasUsers checks if any users exist in the database.
func HasUsers() (bool, error) {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
