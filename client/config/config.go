// Package config loads and persists the TenantZero client configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all client-side settings.
type Config struct {
	ServerURL  string `yaml:"server_url"`  // e.g. "ws://localhost:8080"
	Token      string `yaml:"token"`       // JWT stored after login
	LabsDir    string `yaml:"labs_dir"`    // ~/.tenantzero/labs
	JournalDir string `yaml:"journal_dir"` // ~/.tenantzero/journals
}

// configPath returns the canonical path to the config file.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config: resolve home dir: %w", err)
	}
	return filepath.Join(home, ".tenantzero", "config.yaml"), nil
}

// Load reads ~/.tenantzero/config.yaml.
// If the file is missing, default values are returned with no error.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	home, _ := os.UserHomeDir()
	cfg := &Config{
		ServerURL:  "ws://localhost:8080",
		LabsDir:    filepath.Join(home, ".tenantzero", "labs"),
		JournalDir: filepath.Join(home, ".tenantzero", "journals"),
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}
	return cfg, nil
}

// Save writes the configuration back to ~/.tenantzero/config.yaml,
// creating the parent directory if it does not exist.
func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("config: mkdir %s: %w", filepath.Dir(path), err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("config: write %s: %w", path, err)
	}
	return nil
}