package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const configFileName = ".lazylogcat.json"

// Config represents the application's configuration structure.
// It includes user preferences and session details.
// Prefs holds user preferences for log display.
// Session holds the last used session parameters.
type Config struct {
	Prefs   Prefs   `json:"preferences"`
	Session Session `json:"session"`
}

type Prefs struct {
	Format    string   `json:"log_format,omitempty"`
	Modifiers []string `json:"log_modifiers,omitempty"`
}

type Session struct {
	DeviceID string `json:"device_id,omitempty"`
	Pkg      string `json:"package_name,omitempty"`
	Tag      string `json:"log_tag,omitempty"`
	Txt      string `json:"log_text,omitempty"`
}

func DefaultConfig() Config {
	return Config{
		Prefs: Prefs{
			Format:    "brief",
			Modifiers: []string{"color"},
		},
		Session: Session{
			DeviceID: "",
			Pkg:      "",
			Tag:      "",
			Txt:      "",
		},
	}
}

// Save writes the configuration to disk and returns the file path.
func Save(c *Config) (string, error) {
	j, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal config to JSON", "error", err)
		return "", fmt.Errorf("failed to marshal config to JSON: %w", err)
	}

	err = os.WriteFile(configFileName, j, 0644)
	if err != nil {
		slog.Error("Failed to write config file", "error", err)
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	f, err := os.Open(configFileName)
	if err != nil {
		slog.Error("Failed to open config file for verification", "error", err)
		return "", fmt.Errorf("failed to open config file for verification: %w", err)
	}
	defer f.Close()

	absPath, err := filepath.Abs(f.Name())
	if err != nil {
		slog.Error("Failed to get absolute path of config file", "error", err)
		return "", fmt.Errorf("failed to get absolute path of config file: %w", err)
	}

	return absPath, nil
}

// Load reads the configuration from the provided file.
// If the file is invalid or cannot be read, it returns a default configuration and an error.
func Load(file *os.File) (Config, error) {
	var config Config

	decoder := json.NewDecoder(file)
	err := decoder.Decode(&config)

	if err != nil {
		slog.Error("Failed to decode config file", "error", err)
		return DefaultConfig(), fmt.Errorf("failed to decode config file: %w", err)
	}

	return config, nil
}
