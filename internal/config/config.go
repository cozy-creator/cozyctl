package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
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

// InitViper initializes viper with configuration settings
func InitViper() error {
	// Set config name and type
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Add config path
	configDir, err := ConfigDir()
	if err != nil {
		return err
	}
	viper.AddConfigPath(configDir)

	// Set environment variable prefix and enable automatic env
	viper.SetEnvPrefix("COZY")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("hub_url", "https://api.cozy.art")
	viper.SetDefault("builder_url", "https://builder.cozy.art")

	return nil
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
	// Initialize Viper
	if err := InitViper(); err != nil {
		return nil, err
	}

	// If custom path is specified, use it
	if path != "" {
		viper.SetConfigFile(path)
	}

	// Try to read config file
	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, return error
			configPath, _ := ConfigPath()
			return nil, fmt.Errorf("config file not found: %s (run 'cozyctl login' first)", configPath)
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// Unmarshal config into struct
	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// If values are still empty, apply defaults (environment variables might override)
	if cfg.HubURL == "" {
		cfg.HubURL = viper.GetString("hub_url")
	}
	if cfg.BuilderURL == "" {
		cfg.BuilderURL = viper.GetString("builder_url")
	}

	return cfg, nil
}

// Save saves the config to the specified path, or the default path if empty
func Save(cfg *Config, path string) error {
	// Initialize Viper
	if err := InitViper(); err != nil {
		return err
	}

	// Determine the config path
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

	// Set config values in Viper
	viper.Set("hub_url", cfg.HubURL)
	viper.Set("builder_url", cfg.BuilderURL)
	viper.Set("tenant_id", cfg.TenantID)
	viper.Set("token", cfg.Token)

	// Set the config file path
	viper.SetConfigFile(path)

	// Write config file
	if err := viper.WriteConfig(); err != nil {
		// If file doesn't exist, use SafeWriteConfig
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if err := viper.SafeWriteConfig(); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}
		} else {
			return fmt.Errorf("failed to write config: %w", err)
		}
	}

	// Ensure correct file permissions (0600)
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("failed to set config file permissions: %w", err)
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
