package tests

import (
	"os"
	"testing"

	"github.com/prashanta0234/vpsmyth/internal/deploy"
)

func TestDeployNodeDocker(t *testing.T) {
	// This test requires Docker to be installed and network access to GitHub.
	// We'll use a very small public repo for testing if possible, 
	// or just skip if Docker is not available.
	
	if os.Getenv("SKIP_DOCKER_TEST") == "true" {
		t.Skip("Skipping Docker deployment test")
	}

	appName := "test-node-app"
	repoURL := "https://github.com/heroku/node-js-getting-started" // A standard Node.js sample repo
	port := 5001
	env := map[string]string{
		"NODE_ENV": "development",
	}

	err := deploy.DeployNodeDocker(appName, repoURL, port, env)
	if err != nil {
		t.Fatalf("DeployNodeDocker failed: %v", err)
	}

	// Verify metadata file exists
	metaFile := "deployments/test-node-app.json"
	if _, err := os.Stat(metaFile); os.IsNotExist(err) {
		t.Errorf("Metadata file not found: %s", metaFile)
	}

	// Cleanup (optional, might want to keep for manual inspection)
	os.RemoveAll("deployments/test-node-app")
	os.Remove(metaFile)
}
