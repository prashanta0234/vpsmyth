package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// StopApp stops the Docker container for the given app.
func StopApp(appName string) error {
	sanitizedName := sanitizeAppName(appName)
	cmd := exec.Command("docker", "stop", sanitizedName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}
	return nil
}

// StartApp starts the Docker container for the given app.
func StartApp(appName string) error {
	sanitizedName := sanitizeAppName(appName)
	cmd := exec.Command("docker", "start", sanitizedName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}

// RestartApp restarts the Docker container for the given app.
func RestartApp(appName string) error {
	sanitizedName := sanitizeAppName(appName)
	cmd := exec.Command("docker", "restart", sanitizedName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart container: %w", err)
	}
	return nil
}

// DeleteApp stops, removes the container, and deletes the app's metadata and files.
func DeleteApp(appName string) error {
	sanitizedName := sanitizeAppName(appName)

	// 1. Stop and remove container
	exec.Command("docker", "stop", sanitizedName).Run()
	exec.Command("docker", "rm", sanitizedName).Run()

	// 2. Remove metadata file
	baseDir := "deployments"
	metaFile := filepath.Join(baseDir, sanitizedName+".json")
	os.Remove(metaFile)

	// 3. Remove app directory
	appDir := filepath.Join(baseDir, sanitizedName)
	os.RemoveAll(appDir)

	return nil
}

// GetLogs returns the last 100 lines of logs for the given app.
func GetLogs(appName string) (string, error) {
	sanitizedName := sanitizeAppName(appName)
	out, err := exec.Command("docker", "logs", "--tail", "100", sanitizedName).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}
	return string(out), nil
}

// UpdateAppEnv updates the environment variables for an app and restarts it.
func UpdateAppEnv(appName string, newEnv map[string]string) error {
	sanitizedName := sanitizeAppName(appName)
	baseDir := "deployments"
	metaFile := filepath.Join(baseDir, sanitizedName+".json")

	// 1. Load existing metadata
	data, err := os.ReadFile(metaFile)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	var meta DeploymentMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	// 2. Update env
	meta.Env = newEnv

	// 3. Save metadata
	metaData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if err := os.WriteFile(metaFile, metaData, 0644); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	// 4. Restart container with new env
	// We need the image tag. We can infer it from the sanitized name.
	imageTag := fmt.Sprintf("vpsmyth/%s:latest", sanitizedName)

	// Stop and remove existing container
	exec.Command("docker", "stop", sanitizedName).Run()
	exec.Command("docker", "rm", sanitizedName).Run()

	// Run new container
	runArgs := []string{"run", "-d", "--name", sanitizedName, "--label", "managed-by=vpsmyth", "-p", fmt.Sprintf("%d:%d", meta.Port, meta.Port), "--restart", "always"}
	for k, v := range newEnv {
		runArgs = append(runArgs, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	runArgs = append(runArgs, "-e", fmt.Sprintf("PORT=%d", meta.Port))
	runArgs = append(runArgs, imageTag)

	runCmd := exec.Command("docker", runArgs...)
	output, err := runCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run Docker container: %s: %w", string(output), err)
	}

	// Update container ID in metadata
	meta.ContainerID = string(output[:12])
	metaData, _ = json.MarshalIndent(meta, "", "  ")
	os.WriteFile(metaFile, metaData, 0644)

	return nil
}
