package tests

import (
	"os"
	"testing"

	"github.com/prashanta0234/vpsmyth/internal/db"
)

func TestDatabaseOperations(t *testing.T) {
	dbPath := "test_vpsmyth.db"
	defer os.Remove(dbPath)

	if err := db.InitDB(dbPath); err != nil {
		t.Fatalf("Failed to initialize test DB: %v", err)
	}

	// Test DockerHub Credentials
	err := db.SaveDockerHubCredentials("testuser", "testpass")
	if err != nil {
		t.Errorf("SaveDockerHubCredentials failed: %v", err)
	}

	user, pass, err := db.GetDockerHubCredentials()
	if err != nil {
		t.Errorf("GetDockerHubCredentials failed: %v", err)
	}
	if user != "testuser" || pass != "testpass" {
		t.Errorf("Expected testuser/testpass, got %s/%s", user, pass)
	}

	// Test GitHub Credentials
	err = db.SaveGitHubCredentials("gh_token_123")
	if err != nil {
		t.Errorf("SaveGitHubCredentials failed: %v", err)
	}

	token, err := db.GetGitHubCredentials()
	if err != nil {
		t.Errorf("GetGitHubCredentials failed: %v", err)
	}
	if token != "gh_token_123" {
		t.Errorf("Expected gh_token_123, got %s", token)
	}

	// Test Global Secrets
	err = db.SaveSecret("API_KEY", "secret123")
	if err != nil {
		t.Errorf("SaveSecret failed: %v", err)
	}

	secrets, err := db.GetGlobalSecrets()
	if err != nil {
		t.Errorf("GetGlobalSecrets failed: %v", err)
	}
	if secrets["API_KEY"] != "secret123" {
		t.Errorf("Expected API_KEY=secret123, got %s", secrets["API_KEY"])
	}

	err = db.DeleteSecret("API_KEY")
	if err != nil {
		t.Errorf("DeleteSecret failed: %v", err)
	}

	secrets, _ = db.GetGlobalSecrets()
	if _, exists := secrets["API_KEY"]; exists {
		t.Error("Secret API_KEY should have been deleted")
	}
}
