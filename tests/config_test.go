package tests

import (
	"os"
	"testing"

	"github.com/prashanta0234/vpsmyth/internal/config"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file for testing
	content := `{
		"apps": [
			{
				"name": "test-app",
				"port": 9000,
				"env": {
					"KEY": "VALUE"
				}
			}
		]
	}`
	tmpFile := "test_config.json"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}
	defer os.Remove(tmpFile)

	cfg, err := config.LoadConfig(tmpFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if len(cfg.Apps) != 1 {
		t.Errorf("expected 1 app, got %d", len(cfg.Apps))
	}

	app, ok := cfg.GetAppConfig("test-app")
	if !ok {
		t.Error("failed to get test-app config")
	}

	if app.Port != 9000 {
		t.Errorf("expected port 9000, got %d", app.Port)
	}

	if app.Env["KEY"] != "VALUE" {
		t.Errorf("expected env KEY=VALUE, got %s", app.Env["KEY"])
	}

	_, ok = cfg.GetAppConfig("non-existent")
	if ok {
		t.Error("expected false for non-existent app")
	}
}
