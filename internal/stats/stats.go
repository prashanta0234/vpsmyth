package stats

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// Stats represents the system and application statistics.
type Stats struct {
	CPUUsage   string `json:"cpuUsage"`
	Memory     string `json:"memory"`
	ActiveApps int    `json:"activeApps"`
	Uptime     string `json:"uptime"`
}

// GetStats gathers real-time statistics from the system and Docker.
func GetStats() (Stats, error) {
	var s Stats

	// 1. CPU Usage (simplified for Linux)
	cpu, err := getCPUUsage()
	if err == nil {
		s.CPUUsage = cpu
	} else {
		s.CPUUsage = "N/A"
	}

	// 2. Memory Usage
	mem, err := getMemoryUsage()
	if err == nil {
		s.Memory = mem
	} else {
		s.Memory = "N/A"
	}

	// 3. Active Apps (Docker containers with label managed-by=vpsmyth)
	apps, err := getActiveAppsCount()
	if err == nil {
		s.ActiveApps = apps
	}

	// 4. Uptime
	uptime, err := getUptime()
	if err == nil {
		s.Uptime = uptime
	} else {
		s.Uptime = "N/A"
	}

	return s, nil
}

func getCPUUsage() (string, error) {
	if runtime.GOOS != "linux" {
		return "N/A", nil
	}
	// Use top to get idle CPU and subtract from 100
	out, err := exec.Command("sh", "-c", "top -bn1 | grep \"Cpu(s)\" | sed \"s/.*, *\\([0-9.]*\\)%* id.*/\\1/\" | awk '{print 100 - $1}'").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)) + "%", nil
}

func getMemoryUsage() (string, error) {
	if runtime.GOOS != "linux" {
		return "N/A", nil
	}
	// Use free -h to get used memory
	out, err := exec.Command("sh", "-c", "free -h | grep Mem | awk '{print $3}'").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func getActiveAppsCount() (int, error) {
	// Count containers with label managed-by=vpsmyth
	out, err := exec.Command("sh", "-c", "docker ps -q --filter \"label=managed-by=vpsmyth\" | wc -l").Output()
	if err != nil {
		return 0, err
	}
	count, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, err
	}
	return count, nil
}

func getUptime() (string, error) {
	if runtime.GOOS != "linux" {
		return "N/A", nil
	}
	// Use uptime -p for pretty format
	out, err := exec.Command("uptime", "-p").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(strings.TrimSpace(string(out)), "up "), nil
}
