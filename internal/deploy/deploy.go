package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
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
func DeployNodeDocker(appName string, category string, framework string, repoURL string, port int, env map[string]string) error {
	// Sanitize app name for Docker and file system
	sanitizedName := sanitizeAppName(appName)

	baseDir := "deployments"
	appDir := filepath.Join(baseDir, sanitizedName)
	repoDir := filepath.Join(appDir, "repo")
	metaFile := filepath.Join(baseDir, sanitizedName+".json")

	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	fmt.Printf("Cloning repository: %s\n", repoURL)
	cloneCmd := exec.Command("git", "clone", repoURL, repoDir)
	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err == nil {
		cloneCmd = exec.Command("git", "-C", repoDir, "pull")
	}
	if err := cloneCmd.Run(); err != nil {
		return fmt.Errorf("failed to clone/pull repository: %w", err)
	}

	dockerfilePath := filepath.Join(repoDir, "Dockerfile")
	
	// Check if Dockerfile is tracked by git
	isGitTracked := false
	checkGitCmd := exec.Command("git", "-C", repoDir, "ls-files", "--error-unmatch", "Dockerfile")
	if err := checkGitCmd.Run(); err == nil {
		isGitTracked = true
	}

	// Generate/Overwrite Dockerfile if:
	// - It doesn't exist
	// - OR it's NOT tracked by git (meaning we probably created it)
	// - OR the user explicitly selected a framework (they want our template)
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) || !isGitTracked || framework != "" {
		fmt.Printf("Generating Dockerfile for framework: %s (%s)\n", framework, category)
		dockerfileContent := generateSmartDockerfile(repoDir, port, category, framework)
		if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
			return fmt.Errorf("failed to create Dockerfile: %w", err)
		}
	}

	dockerIgnorePath := filepath.Join(repoDir, ".dockerignore")
	if _, err := os.Stat(dockerIgnorePath); os.IsNotExist(err) {
		dockerIgnoreContent := "node_modules\n.next\ndist\nbuild\n.git\n"
		os.WriteFile(dockerIgnorePath, []byte(dockerIgnoreContent), 0644)
	}

	imageTag := fmt.Sprintf("vpsmyth/%s:latest", sanitizedName)
	fmt.Printf("Building Docker image: %s\n", imageTag)
	buildCmd := exec.Command("docker", "build", "-t", imageTag, repoDir)
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	// 5. Run Docker container
	fmt.Printf("Running Docker container for: %s (sanitized: %s)\n", appName, sanitizedName)
	// Stop and remove existing container if it exists
	exec.Command("docker", "stop", sanitizedName).Run()
	exec.Command("docker", "rm", sanitizedName).Run()

	runArgs := []string{"run", "-d", "--name", sanitizedName, "--label", "managed-by=vpsmyth", "-p", fmt.Sprintf("%d:%d", port, port), "--restart", "always"}
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

// sanitizeAppName converts a string to a Docker-compatible name.
func sanitizeAppName(name string) string {
	// Replace spaces with hyphens
	sanitized := strings.ReplaceAll(name, " ", "-")
	// Convert to lowercase
	sanitized = strings.ToLower(sanitized)
	// Remove non-alphanumeric/hyphen characters
	reg := regexp.MustCompile("[^a-z0-9-]")
	sanitized = reg.ReplaceAllString(sanitized, "")
	return sanitized
}

// generateSmartDockerfile analyzes the repository and returns a suitable Dockerfile.
func generateSmartDockerfile(repoDir string, port int, category string, framework string) string {
	// If user explicitly selected a framework, use it
	if framework != "" {
		switch framework {
		case "nextjs":
			return `FROM node:18-alpine
WORKDIR /app
RUN apk add --no-cache libc6-compat
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build
EXPOSE ` + fmt.Sprint(port) + `
ENV PORT ` + fmt.Sprint(port) + `
CMD ["npm", "start"]
`
		case "react":
			return `FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build
RUN npm install -g serve
EXPOSE ` + fmt.Sprint(port) + `
CMD ["serve", "-s", "dist", "-p", "` + fmt.Sprint(port) + `"]
`
		case "html":
			return `FROM pierotofy/static-base
COPY . /public
EXPOSE ` + fmt.Sprint(port) + `
CMD ["-p", "` + fmt.Sprint(port) + `"]
`
		case "nestjs":
			return `FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build
EXPOSE ` + fmt.Sprint(port) + `
CMD ["npm", "run", "start:prod"]
`
		case "express", "nodejs":
			return `FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
EXPOSE ` + fmt.Sprint(port) + `
CMD ["npm", "start"]
`
		}
	}

	// Fallback to auto-detection if framework is empty
	packageJSONPath := filepath.Join(repoDir, "package.json")
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		// No package.json, assume static HTML
		return `FROM pierotofy/static-base
COPY . /public
EXPOSE ` + fmt.Sprint(port) + `
CMD ["-p", "` + fmt.Sprint(port) + `"]
`
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
		Scripts         map[string]string `json:"scripts"`
	}
	json.Unmarshal(data, &pkg)

	isNext := pkg.Dependencies["next"] != "" || pkg.DevDependencies["next"] != ""
	isVite := pkg.Dependencies["vite"] != "" || pkg.DevDependencies["vite"] != ""
	isReact := pkg.Dependencies["react-scripts"] != "" || pkg.DevDependencies["react-scripts"] != ""
	isNest := pkg.Dependencies["@nestjs/core"] != ""

	if isNext {
		return generateSmartDockerfile(repoDir, port, "frontend", "nextjs")
	}
	if isNest {
		return generateSmartDockerfile(repoDir, port, "backend", "nestjs")
	}
	if isVite || isReact {
		return generateSmartDockerfile(repoDir, port, "frontend", "react")
	}

	return generateSmartDockerfile(repoDir, port, "backend", "nodejs")
}
