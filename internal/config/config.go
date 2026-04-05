// Package config provides read/write access to the devforge configuration file.
// Config path: ~/.config/devforge/config.json
// Override: DEV_FORGE_CONFIG environment variable.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds all user-configurable settings for devforge.
type Config struct {
	GeminiAPIKey string `json:"gemini_api_key"`
	ImageModel   string `json:"image_model"` // default: gemini-2.5-flash-image
}

// Path resolves the config file path from the environment or the default location.
// It checks Homebrew-specific locations first, then falls back to the standard
// XDG_CONFIG_HOME location (~/.config/devforge/config.json).
func Path() string {
	if p := os.Getenv("DEV_FORGE_CONFIG"); p != "" {
		return p
	}

	// Homebrew-specific config paths (checked first for existing installations)
	homebrewPaths := []string{
		"/home/linuxbrew/.linuxbrew/etc/devforge/config.json", // Linuxbrew
		"/opt/homebrew/etc/devforge/config.json",              // macOS ARM
		"/usr/local/etc/devforge/config.json",                 // macOS Intel
	}
	for _, p := range homebrewPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Standard XDG_CONFIG_HOME location
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		path := filepath.Join(xdgConfig, "devforge", "config.json")
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(home, ".config", "devforge", "config.json")
}

// Load reads and parses the config file.
// Returns an empty Config{} (with defaults) if the file does not exist.
func Load() (*Config, error) {
	p := Path()
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{
				ImageModel: "gemini-2.5-flash-image",
			}, nil
		}
		return nil, err
	}

	cfg := &Config{
		ImageModel: "gemini-2.5-flash-image",
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	// Apply defaults for empty fields
	if cfg.ImageModel == "" {
		cfg.ImageModel = "gemini-2.5-flash-image"
	}
	return cfg, nil
}

// Save writes the config to disk with 0600 permissions.
// Creates the config directory if needed.
func Save(cfg *Config) error {
	p := Path()
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0600)
}
