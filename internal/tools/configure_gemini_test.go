package tools_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"dev-forge-mcp/internal/config"
	"dev-forge-mcp/internal/tools"
)

func TestConfigureGemini_WritesConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	t.Setenv("DEV_FORGE_CONFIG", cfgPath)

	srv := &tools.Server{}

	var hotReloaded string
	result := srv.ConfigureGemini(context.Background(),
		tools.ConfigureGeminiInput{APIKey: "test-api-key-xyz"},
		func(key string) { hotReloaded = key },
	)

	var out tools.ConfigureGeminiOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.Status != "saved" {
		t.Errorf("expected status=saved, got %q", out.Status)
	}
	if out.ConfigPath == "" {
		t.Error("expected non-empty config_path")
	}

	// Verify the key was hot-reloaded
	if hotReloaded != "test-api-key-xyz" {
		t.Errorf("expected hot-reload with key, got %q", hotReloaded)
	}

	// Verify the file was written with correct key
	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load after configure: %v", err)
	}
	if loaded.GeminiAPIKey != "test-api-key-xyz" {
		t.Errorf("expected key in config, got %q", loaded.GeminiAPIKey)
	}
}

func TestConfigureGemini_WritesWithCorrectPermissions(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	t.Setenv("DEV_FORGE_CONFIG", cfgPath)

	srv := &tools.Server{}

	srv.ConfigureGemini(context.Background(),
		tools.ConfigureGeminiInput{APIKey: "perm-test-key"},
		nil,
	)

	info, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600 permissions, got %o", info.Mode().Perm())
	}
}

func TestConfigureGemini_EmptyAPIKey_ReturnsError(t *testing.T) {
	srv := &tools.Server{}

	result := srv.ConfigureGemini(context.Background(),
		tools.ConfigureGeminiInput{APIKey: ""},
		nil,
	)

	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := errOut["error"]; !ok {
		t.Errorf("expected error for empty API key, got: %s", result)
	}
}
