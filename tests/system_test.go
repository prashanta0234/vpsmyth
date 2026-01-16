package tests

import (
	"testing"

	"github.com/prashanta0234/vpsmyth/internal/system"
)

func TestSystemStatus(t *testing.T) {
	status := system.GetSystemStatus()
	
	// We can't guarantee tools are installed, but we can check if the logic returns something
	// If it's installed, version shouldn't be empty
	if status.Docker.Installed && status.Docker.Version == "" {
		t.Error("Docker is installed but version is empty")
	}
	if status.Node.Installed && status.Node.Version == "" {
		t.Error("Node is installed but version is empty")
	}
	if status.Go.Installed && status.Go.Version == "" {
		t.Error("Go is installed but version is empty")
	}
}

func TestListContainers(t *testing.T) {
	containers, err := system.ListContainers()
	if err != nil {
		t.Fatalf("ListContainers failed: %v", err)
	}

	// Even if no containers are running, it should return an empty slice, not nil
	if containers == nil {
		t.Error("ListContainers returned nil slice")
	}
}
