package system

import (
	"fmt"
	"os/exec"
	"strings"
)

// ToolStatus represents the status of a system tool.
type ToolStatus struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
}

// SystemStatus represents the status of all managed system tools.
type SystemStatus struct {
	Docker ToolStatus `json:"docker"`
	Node   ToolStatus `json:"node"`
	Go     ToolStatus `json:"go"`
}

// GetSystemStatus checks the installation status and versions of system tools.
func GetSystemStatus() SystemStatus {
	return SystemStatus{
		Docker: checkTool("docker", "--version"),
		Node:   checkTool("node", "--version"),
		Go:     checkTool("go", "version"),
	}
}

func checkTool(name string, versionArg string) ToolStatus {
	_, err := exec.LookPath(name)
	if err != nil {
		return ToolStatus{Installed: false, Version: ""}
	}

	out, err := exec.Command(name, versionArg).Output()
	if err != nil {
		return ToolStatus{Installed: true, Version: "Unknown"}
	}

	version := strings.TrimSpace(string(out))
	// Clean up version strings (e.g., "Docker version 24.0.7, build 24.0.7-0ubuntu1~22.04.1")
	if name == "docker" {
		parts := strings.Split(version, " ")
		if len(parts) >= 3 {
			version = parts[2]
		}
	} else if name == "node" {
		version = strings.TrimPrefix(version, "v")
	} else if name == "go" {
		parts := strings.Split(version, " ")
		if len(parts) >= 3 {
			version = strings.TrimPrefix(parts[2], "go")
		}
	}

	return ToolStatus{Installed: true, Version: version}
}

// InstallNode installs Node.js on the host system using the NodeSource script.
func InstallNode() error {
	// 1. Download and run NodeSource setup script (Node.js 20.x)
	setupCmd := exec.Command("sh", "-c", "curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -")
	if err := setupCmd.Run(); err != nil {
		return fmt.Errorf("failed to run NodeSource setup script: %w", err)
	}

	// 2. Install Node.js
	installCmd := exec.Command("sudo", "apt-get", "install", "-y", "nodejs")
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install nodejs: %w", err)
	}

	return nil
}

// InstallDocker installs Docker on the host system.
func InstallDocker() error {
	setupCmd := exec.Command("sh", "-c", "curl -fsSL https://get.docker.com -o get-docker.sh && sudo sh get-docker.sh && rm get-docker.sh")
	if err := setupCmd.Run(); err != nil {
		return fmt.Errorf("failed to install Docker: %w", err)
	}
	return nil
}

// InstallGo installs Go on the host system.
func InstallGo() error {
	goVersion := "1.21.5"
	installCmd := exec.Command("sh", "-c", fmt.Sprintf("curl -LO https://go.dev/dl/go%s.linux-amd64.tar.gz && sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go%s.linux-amd64.tar.gz && rm go%s.linux-amd64.tar.gz", goVersion, goVersion, goVersion))
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install Go: %w", err)
	}
	return nil
}
