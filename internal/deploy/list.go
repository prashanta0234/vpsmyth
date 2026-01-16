package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ListApps returns a list of all deployed applications by inspecting Docker containers and metadata files.
func ListApps() ([]DeploymentMetadata, error) {
	// 1. Get all containers managed by VPSMyth
	out, err := exec.Command("docker", "ps", "-a", "--filter", "label=managed-by=vpsmyth", "--format", "{{.Names}}|{{.ID}}|{{.Status}}|{{.Ports}}").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list docker containers: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var apps []DeploymentMetadata
	baseDir := "deployments"

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}

		containerName := parts[0]
		containerID := parts[1]
		status := parts[2]
		portsStr := parts[3] // e.g., "0.0.0.0:3000->3000/tcp" or "3000/tcp"

		// 2. Try to find metadata file
		metaFile := filepath.Join(baseDir, containerName+".json")
		var meta DeploymentMetadata
		data, err := os.ReadFile(metaFile)
		if err == nil {
			// Metadata exists, use it
			if err := json.Unmarshal(data, &meta); err != nil {
				fmt.Printf("Warning: failed to parse metadata for %s: %v\n", containerName, err)
			}
		}

		// 3. Fill in missing info from Docker if metadata is missing or incomplete
		if meta.AppName == "" {
			meta.AppName = containerName // Fallback to container name
		}
		meta.ContainerID = containerID
		meta.Status = status

		// Parse port from Docker ports string if missing from metadata
		if meta.Port == 0 && portsStr != "" {
			// Example portsStr: "0.0.0.0:3000->3000/tcp"
			// We want the host port (3000)
			re := regexp.MustCompile(`:(\d+)->`)
			match := re.FindStringSubmatch(portsStr)
			if len(match) > 1 {
				p, _ := strconv.Atoi(match[1])
				meta.Port = p
			}
		}

		apps = append(apps, meta)
	}

	return apps, nil
}
