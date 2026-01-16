package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ListApps returns a list of all deployed applications by reading metadata files.
func ListApps() ([]DeploymentMetadata, error) {
	baseDir := "deployments"
	var apps []DeploymentMetadata

	// Ensure the directory exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return apps, nil
	}

	files, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read deployments directory: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			filePath := filepath.Join(baseDir, file.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("Warning: failed to read metadata file %s: %v\n", filePath, err)
				continue
			}

			var meta DeploymentMetadata
			if err := json.Unmarshal(data, &meta); err != nil {
				fmt.Printf("Warning: failed to parse metadata file %s: %v\n", filePath, err)
				continue
			}
			apps = append(apps, meta)
		}
	}

	return apps, nil
}
