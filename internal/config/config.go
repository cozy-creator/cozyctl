package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds CLI configuration
type Config struct {
	// HubURL is the cozy-hub API URL for authentication
	HubURL string `yaml:"hub_url"`

	// BuilderURL is the gen-builder API URL for builds
	BuilderURL string `yaml:"builder_url"`

	// TenantID is the authenticated tenant ID
	TenantID string `yaml:"tenant_id"`

	// Token is the API token from login
	Token string `yaml:"token"`
}

// Default returns a config with default values
func Default() *Config {
	return &Config{
		HubURL:     "https://api.cozy.art",
		BuilderURL: "https://builder.cozy.art",
	}
}

// ConfigDir returns the config directory path (~/.cozy)
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".cozy"), nil
}

// ConfigPath returns the config file path (~/.cozy/config.yaml)
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Load loads the config from the specified path, or the default path if empty
func Load(path string) (*Config, error) {
	if path == "" {
		var err error
		path, err = ConfigPath()
		if err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s (run 'cozy login' first)", path)
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}

// Save saves the config to the specified path, or the default path if empty
func Save(cfg *Config, path string) error {
	if path == "" {
		var err error
		path, err = ConfigPath()
		if err != nil {
			return err
		}
	}

	// Ensure config directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Validate checks that required fields are set
func (c *Config) Validate() error {
	if c.Token == "" {
		return fmt.Errorf("not logged in (run 'cozy login' first)")
	}
	if c.TenantID == "" {
		return fmt.Errorf("tenant_id not set in config")
	}
	return nil
}
