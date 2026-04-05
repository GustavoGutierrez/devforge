package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"dev-forge-mcp/internal/config"
)

func TestLoad_MissingFile_ReturnsEmptyConfig(t *testing.T) {
	// Point to a non-existent file
	t.Setenv("DEV_FORGE_CONFIG", filepath.Join(t.TempDir(), "nonexistent.json"))

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load returned error for missing file: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load returned nil config")
	}
	if cfg.ImageModel != "gemini-2.5-flash-image" {
		t.Errorf("default ImageModel wrong: %s", cfg.ImageModel)
	}
}

func TestSave_WritesWithCorrectPermissions(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	t.Setenv("DEV_FORGE_CONFIG", cfgPath)

	cfg := &config.Config{
		GeminiAPIKey: "test-key",
		ImageModel:   "gemini-2.5-flash-image",
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	// Check file exists
	info, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	// Check 0600 permissions
	mode := info.Mode().Perm()
	if mode != 0600 {
		t.Errorf("expected 0600 permissions, got %o", mode)
	}
}

func TestLoad_AfterSave_ReturnsKey(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	t.Setenv("DEV_FORGE_CONFIG", cfgPath)

	cfg := &config.Config{
		GeminiAPIKey: "my-gemini-key-123",
		ImageModel:   "gemini-2.5-flash-image",
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.GeminiAPIKey != "my-gemini-key-123" {
		t.Errorf("expected key %q, got %q", "my-gemini-key-123", loaded.GeminiAPIKey)
	}
}

func TestPath_EnvOverride(t *testing.T) {
	expected := "/tmp/my-custom-config.json"
	t.Setenv("DEV_FORGE_CONFIG", expected)
	if got := config.Path(); got != expected {
		t.Errorf("Path() = %q, want %q", got, expected)
	}
}

func TestPath_DefaultLocation(t *testing.T) {
	t.Setenv("DEV_FORGE_CONFIG", "")
	p := config.Path()
	if p == "" {
		t.Error("Path() returned empty string for default location")
	}
	// Should contain ".config/dev-forge/config.json"
	if filepath.Base(p) != "config.json" {
		t.Errorf("expected filename config.json, got %s", filepath.Base(p))
	}
}
