package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// DeploymentMetadata stores information about a deployed application.
type DeploymentMetadata struct {
	AppName     string            `json:"app_name"`
	ContainerID string            `json:"container_id"`
	Port        int               `json:"port"`
	Status      string            `json:"status"`
	Env         map[string]string `json:"env"`
}

// DeployNodeDocker deploys a Node.js application using Docker.
func DeployNodeDocker(appName string, repoURL string, port int, env map[string]string) error {
	// 1. Setup directories
	baseDir := "deployments"
	appDir := filepath.Join(baseDir, appName)
	repoDir := filepath.Join(appDir, "repo")
	metaFile := filepath.Join(baseDir, appName+".json")

	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// 2. Clone repository
	fmt.Printf("Cloning repository: %s\n", repoURL)
	cloneCmd := exec.Command("git", "clone", repoURL, repoDir)
	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err == nil {
		// Repo already exists, pull latest
		cloneCmd = exec.Command("git", "-C", repoDir, "pull")
	}
	if err := cloneCmd.Run(); err != nil {
		return fmt.Errorf("failed to clone/pull repository: %w", err)
	}

	// 3. Check/Create Dockerfile
	dockerfilePath := filepath.Join(repoDir, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		fmt.Println("Dockerfile missing, generating a basic one...")
		dockerfileContent := `FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
EXPOSE ` + fmt.Sprint(port) + `
CMD ["npm", "start"]
`
		if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
			return fmt.Errorf("failed to create Dockerfile: %w", err)
		}
	}

	// 4. Build Docker image
	imageTag := fmt.Sprintf("vpsmyth/%s:latest", appName)
	fmt.Printf("Building Docker image: %s\n", imageTag)
	buildCmd := exec.Command("docker", "build", "-t", imageTag, repoDir)
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	// 5. Run Docker container
	fmt.Printf("Running Docker container for: %s\n", appName)
	// Stop and remove existing container if it exists
	exec.Command("docker", "stop", appName).Run()
	exec.Command("docker", "rm", appName).Run()

	runArgs := []string{"run", "-d", "--name", appName, "-p", fmt.Sprintf("%d:%d", port, port), "--restart", "always"}
	for k, v := range env {
		runArgs = append(runArgs, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	runArgs = append(runArgs, "-e", fmt.Sprintf("PORT=%d", port))
	runArgs = append(runArgs, imageTag)

	runCmd := exec.Command("docker", runArgs...)
	output, err := runCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run Docker container: %s: %w", string(output), err)
	}
	containerID := string(output[:12]) // Get short container ID

	// 6. Store metadata
	meta := DeploymentMetadata{
		AppName:     appName,
		ContainerID: containerID,
		Port:        port,
		Status:      "running",
		Env:         env,
	}
	metaData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if err := os.WriteFile(metaFile, metaData, 0644); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	fmt.Printf("Successfully deployed %s (Container ID: %s)\n", appName, containerID)
	return nil
}
