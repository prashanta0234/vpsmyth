package system

import (
	"fmt"
	"os/exec"
	"strings"
)

// Container represents a Docker container.
type Container struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Image   string `json:"image"`
	Status  string `json:"status"`
	Ports   string `json:"ports"`
	Running bool   `json:"running"`
}

// ListContainers returns a list of all Docker containers on the system.
func ListContainers() ([]Container, error) {
	// Format: ID|Names|Image|Status|Ports|State
	out, err := exec.Command("docker", "ps", "-a", "--format", "{{.ID}}|{{.Names}}|{{.Image}}|{{.Status}}|{{.Ports}}|{{.State}}").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var containers []Container
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 6 {
			continue
		}

		containers = append(containers, Container{
			ID:      parts[0],
			Name:    parts[1],
			Image:   parts[2],
			Status:  parts[3],
			Ports:   parts[4],
			Running: parts[5] == "running",
		})
	}

	return containers, nil
}

// StartContainer starts a Docker container by ID or Name.
func StartContainer(id string) error {
	if err := exec.Command("docker", "start", id).Run(); err != nil {
		return fmt.Errorf("failed to start container %s: %w", id, err)
	}
	return nil
}

// StopContainer stops a Docker container by ID or Name.
func StopContainer(id string) error {
	if err := exec.Command("docker", "stop", id).Run(); err != nil {
		return fmt.Errorf("failed to stop container %s: %w", id, err)
	}
	return nil
}

// RestartContainer restarts a Docker container by ID or Name.
func RestartContainer(id string) error {
	if err := exec.Command("docker", "restart", id).Run(); err != nil {
		return fmt.Errorf("failed to restart container %s: %w", id, err)
	}
	return nil
}

// DeleteContainer removes a Docker container by ID or Name.
func DeleteContainer(id string) error {
	// Force remove to handle running containers
	if err := exec.Command("docker", "rm", "-f", id).Run(); err != nil {
		return fmt.Errorf("failed to delete container %s: %w", id, err)
	}
	return nil
}

// GetContainerLogs returns the last 100 lines of logs for a container.
func GetContainerLogs(id string) (string, error) {
	out, err := exec.Command("docker", "logs", "--tail", "100", id).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get logs for container %s: %w", id, err)
	}
	return string(out), nil
}
