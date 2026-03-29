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
	GeminiAPIKey   string `json:"gemini_api_key"`
	OllamaURL      string `json:"ollama_url"`      // default: http://localhost:11434
	EmbeddingModel string `json:"embedding_model"` // default: nomic-embed-text (768-dim)
	ImageModel     string `json:"image_model"`     // default: gemini-2.5-flash-image
}

// Path resolves the config file path from the environment or the default location.
func Path() string {
	if p := os.Getenv("DEV_FORGE_CONFIG"); p != "" {
		return p
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
				OllamaURL:      "http://localhost:11434",
				EmbeddingModel: "nomic-embed-text",
				ImageModel:     "gemini-2.5-flash-image",
			}, nil
		}
		return nil, err
	}

	cfg := &Config{
		OllamaURL:      "http://localhost:11434",
		EmbeddingModel: "nomic-embed-text",
		ImageModel:     "gemini-2.5-flash-image",
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	// Apply defaults for empty fields
	if cfg.OllamaURL == "" {
		cfg.OllamaURL = "http://localhost:11434"
	}
	if cfg.EmbeddingModel == "" {
		cfg.EmbeddingModel = "nomic-embed-text"
	}
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
