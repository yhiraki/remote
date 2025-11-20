package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfig_Load(t *testing.T) {
	// Helper to create a dummy config file
	createConfigFile := func(t *testing.T, dir, name, content string) string {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	tmpDir, err := os.MkdirTemp("", "remote_config_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Change CWD to temp dir for findConfigFile tests
	originalWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalWd)

	t.Run("load valid config", func(t *testing.T) {
		configName := ".remoterc.json"
		configContent := `{"hostname": "example.com", "excludeFiles": ["*.log"]}`
		createConfigFile(t, tmpDir, configName, configContent)

		cfg, err := New()
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		if err := cfg.Load(configName); err != nil {
			t.Errorf("Config.Load() error = %v", err)
		}

		if cfg.Hostname != "example.com" {
			t.Errorf("Config.Hostname = %v, want %v", cfg.Hostname, "example.com")
		}
		if !reflect.DeepEqual(cfg.ExcludeFiles, []string{"*.log"}) {
			t.Errorf("Config.ExcludeFiles = %v, want %v", cfg.ExcludeFiles, []string{"*.log"})
		}
	})

	t.Run("config not found", func(t *testing.T) {
		// Ensure no config exists in current temp dir
		cfg, err := New()
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		// We expect Load to fail or behave specifically if global config also missing.
		// Since New() sets ConfigDir to user home based, and we can't easily mock user home without more refactoring,
		// we will focus on the fact that local config is missing.
		// However, Load implementation falls back to global config.
		// If global config doesn't exist, Open fails.
		
		// For this test, we assume global config at ~/.config/remote/.remoterc.json likely doesn't exist or we can't control it.
		// So we expect an error.
		if err := cfg.Load("nonexistent.json"); err == nil {
			t.Error("Config.Load() expected error for nonexistent file, got nil")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		configName := "invalid.json"
		createConfigFile(t, tmpDir, configName, `{invalid-json`)

		cfg, err := New()
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		if err := cfg.Load(configName); err == nil {
			t.Error("Config.Load() expected error for invalid json, got nil")
		}
	})
}

