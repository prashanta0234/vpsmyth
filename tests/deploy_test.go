package tests

import (
	"os"
	"os/exec"
	"testing"

	"github.com/prashanta0234/vpsmyth/internal/deploy"
)

func TestDeployNodeDocker(t *testing.T) {
	if os.Getenv("SKIP_DOCKER_TEST") == "true" {
		t.Skip("Skipping Docker deployment test")
	}

	appName := "test-node-app"
	category := "backend"
	framework := "nodejs"
	repoURL := "https://github.com/heroku/node-js-getting-started"
	port := 5001
	env := map[string]string{
		"NODE_ENV": "development",
	}

	err := deploy.DeployNodeDocker(appName, category, framework, repoURL, port, env)
	if err != nil {
		t.Fatalf("DeployNodeDocker failed: %v", err)
	}

	// Verify metadata file exists
	metaFile := "deployments/test-node-app.json"
	if _, err := os.Stat(metaFile); os.IsNotExist(err) {
		t.Errorf("Metadata file not found: %s", metaFile)
	}

	// Cleanup
	os.RemoveAll("deployments/test-node-app")
	os.Remove(metaFile)
}

func TestDeployFromImage(t *testing.T) {
	if os.Getenv("SKIP_DOCKER_TEST") == "true" {
		t.Skip("Skipping Docker image test")
	}

	appName := "test-nginx-image"
	imageName := "nginx:stable-alpine"
	port := 8081
	env := map[string]string{"TEST_VAR": "true"}

	err := deploy.DeployFromImage(appName, imageName, port, env)
	if err != nil {
		t.Fatalf("DeployFromImage failed: %v", err)
	}

	// Verify metadata
	metaFile := "deployments/test-nginx-image.json"
	if _, err := os.Stat(metaFile); os.IsNotExist(err) {
		t.Errorf("Metadata file not found: %s", metaFile)
	}

	// Cleanup
	exec.Command("docker", "stop", appName).Run()
	exec.Command("docker", "rm", appName).Run()
	os.Remove(metaFile)
}
