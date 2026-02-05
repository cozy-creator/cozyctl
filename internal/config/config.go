package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// DefaultConfig points to the currently active name+profile
// The place is ~/.cozy/default/config.yaml
type DefaultConfig struct {
	CurrentName    string `yaml:"current_name" mapstructure:"current_name"`
	CurrentProfile string `yaml:"current_profile" mapstructure:"current_profile"`
}

// ProfileConfig holds the complete configuration for a name+profile
type ProfileConfig struct {
	CurrentName    string      `yaml:"current_name" mapstructure:"current_name"`
	CurrentProfile string      `yaml:"current_profile" mapstructure:"current_profile"`
	Config         *ConfigData `yaml:"config" mapstructure:"config"`
}

// ConfigData holds the actual configuration values
type ConfigData struct {
	HubURL          string `yaml:"hub_url" mapstructure:"hub_url"`
	BuilderURL      string `yaml:"builder_url" mapstructure:"builder_url"`
	OrchestratorURL string `yaml:"orchestrator_url" mapstructure:"orchestrator_url"`
	TenantID        string `yaml:"tenant_id" mapstructure:"tenant_id"`
	Token           string `yaml:"token" mapstructure:"token"`
	RefreshToken    string `yaml:"refresh_token,omitempty" mapstructure:"refresh_token"`
}

// BaseDir returns the base config directory (~/.cozy)
func BaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".cozy"), nil
}

// DefaultConfigPath returns the path to the default pointer config
func DefaultConfigPath() (string, error) {
	base, err := BaseDir()
	if err != nil {
		return "", err
	}

	// ~/.cozy/default/config.yaml
	return filepath.Join(base, "default", "config.yaml"), nil
}

// ProfileDir returns the directory for a name+profile
func ProfileDir(name, profile string) (string, error) {
	base, err := BaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, name, profile), nil
}

// ProfileConfigPath returns the config file path for a name+profile
func ProfileConfigPath(name, profile string) (string, error) {
	dir, err := ProfileDir(name, profile)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// GetDefaultConfig reads the default pointer config
func GetDefaultConfig() (*DefaultConfig, error) {
	configPath, err := DefaultConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config file
		defaults := &DefaultConfig{
			CurrentName:    "default",
			CurrentProfile: "default",
		}
		if err := SaveDefaultConfig(defaults.CurrentName, defaults.CurrentProfile); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return defaults, nil
	}

	// Create Viper instance
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read default config: %w", err)
	}

	cfg := &DefaultConfig{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse default config: %w", err)
	}

	// Set defaults if empty
	if cfg.CurrentName == "" {
		cfg.CurrentName = "default"
	}
	if cfg.CurrentProfile == "" {
		cfg.CurrentProfile = "default"
	}

	return cfg, nil
}

// SaveDefaultConfig saves the default pointer config
func SaveDefaultConfig(name, profile string) error {
	configPath, err := DefaultConfigPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create default config directory: %w", err)
	}

	// Create Viper instance
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Set values
	v.Set("current_name", name)
	v.Set("current_profile", profile)

	// Write config using WriteConfigAs which handles both new and existing files
	if err := v.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to write default config: %w", err)
	}

	// Ensure correct permissions
	if err := os.Chmod(configPath, 0600); err != nil {
		return fmt.Errorf("failed to set config file permissions: %w", err)
	}

	return nil
}

// GetProfileConfig reads a profile config
func GetProfileConfig(name, profile string) (*ProfileConfig, error) {
	configPath, err := ProfileConfigPath(name, profile)
	if err != nil {
		return nil, err
	}

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("profile '%s/%s' not found (run 'cozyctl login --name %s --profile %s' first)", name, profile, name, profile)
	}

	// Create Viper instance
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Set environment variable support
	v.SetEnvPrefix("COZY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Set defaults
	v.SetDefault("config.hub_url", "http://localhost:3001")
	v.SetDefault("config.builder_url", "http://localhost:3001")
	v.SetDefault("config.orchestrator_url", "http://localhost:8090")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read profile config: %w", err)
	}

	cfg := &ProfileConfig{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse profile config: %w", err)
	}

	// Apply environment variable overrides
	if cfg.Config != nil {
		if v.IsSet("hub_url") {
			cfg.Config.HubURL = v.GetString("hub_url")
		}
		if v.IsSet("builder_url") {
			cfg.Config.BuilderURL = v.GetString("builder_url")
		}
		if v.IsSet("orchestrator_url") {
			cfg.Config.OrchestratorURL = v.GetString("orchestrator_url")
		}
		if v.IsSet("token") {
			cfg.Config.Token = v.GetString("token")
		}
		if v.IsSet("tenant_id") {
			cfg.Config.TenantID = v.GetString("tenant_id")
		}
		if v.IsSet("refresh_token") {
			cfg.Config.RefreshToken = v.GetString("refresh_token")
		}
	}

	return cfg, nil
}

// SaveProfileConfig saves a profile config
func SaveProfileConfig(name, profile string, cfg *ProfileConfig) error {
	configPath, err := ProfileConfigPath(name, profile)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}

	// Create Viper instance
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Set values
	v.Set("current_name", cfg.CurrentName)
	v.Set("current_profile", cfg.CurrentProfile)
	if cfg.Config != nil {
		v.Set("config.hub_url", cfg.Config.HubURL)
		v.Set("config.builder_url", cfg.Config.BuilderURL)
		v.Set("config.orchestrator_url", cfg.Config.OrchestratorURL)
		v.Set("config.tenant_id", cfg.Config.TenantID)
		v.Set("config.token", cfg.Config.Token)
		if cfg.Config.RefreshToken != "" {
			v.Set("config.refresh_token", cfg.Config.RefreshToken)
		}
	}

	// Write config using WriteConfigAs which handles both new and existing files
	if err := v.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to write profile config: %w", err)
	}

	// Ensure correct permissions
	if err := os.Chmod(configPath, 0600); err != nil {
		return fmt.Errorf("failed to set config file permissions: %w", err)
	}

	return nil
}

// ProfileExists checks if a profile exists
func ProfileExists(name, profile string) bool {
	configPath, err := ProfileConfigPath(name, profile)
	if err != nil {
		return false
	}
	_, err = os.Stat(configPath)
	return err == nil
}

// ListAllProfiles scans the directory structure and returns all profiles
func ListAllProfiles() ([]struct{ Name, Profile string }, error) {
	base, err := BaseDir()
	if err != nil {
		return nil, err
	}

	var profiles []struct{ Name, Profile string }

	// Read all name directories
	nameEntries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return profiles, nil
		}
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	for _, nameEntry := range nameEntries {
		if !nameEntry.IsDir() || nameEntry.Name() == "default" {
			continue
		}

		namePath := filepath.Join(base, nameEntry.Name())
		profileEntries, err := os.ReadDir(namePath)
		if err != nil {
			continue
		}

		for _, profileEntry := range profileEntries {
			if !profileEntry.IsDir() {
				continue
			}

			// Check if config.yaml exists
			configPath := filepath.Join(namePath, profileEntry.Name(), "config.yaml")
			if _, err := os.Stat(configPath); err == nil {
				profiles = append(profiles, struct{ Name, Profile string }{
					Name:    nameEntry.Name(),
					Profile: profileEntry.Name(),
				})
			}
		}
	}

	return profiles, nil
}

// DeleteProfile removes a profile directory
func DeleteProfile(name, profile string) error {
	// Prevent deletion of default/default
	if name == "default" && profile == "default" {
		return fmt.Errorf("cannot delete default/default profile")
	}

	dir, err := ProfileDir(name, profile)
	if err != nil {
		return err
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("profile '%s/%s' does not exist", name, profile)
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	return nil
}

// ImportConfigFile imports an external config file
func ImportConfigFile(sourceFile, name, profile string) (*ProfileConfig, error) {
	// Create Viper instance to read source file
	v := viper.New()
	v.SetConfigFile(sourceFile)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Try to unmarshal as ConfigData first (flat structure)
	configData := &ConfigData{}
	if err := v.Unmarshal(configData); err == nil && configData.Token != "" {
		// Successfully unmarshaled as flat config
		return &ProfileConfig{
			CurrentName:    name,
			CurrentProfile: profile,
			Config:         configData,
		}, nil
	}

	// Try to unmarshal as ProfileConfig (nested structure)
	profileCfg := &ProfileConfig{}
	if err := v.Unmarshal(profileCfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Update name and profile
	profileCfg.CurrentName = name
	profileCfg.CurrentProfile = profile

	return profileCfg, nil
}

// Validate checks that required fields are set
func (c *ConfigData) Validate() error {
	if c.Token == "" {
		return fmt.Errorf("not logged in (run 'cozyctl login' first)")
	}
	if c.TenantID == "" {
		return fmt.Errorf("tenant_id not set in config")
	}
	return nil
}

// DefaultConfigData returns default config values
func DefaultConfigData() *ConfigData {
	return &ConfigData{
		HubURL:          "http://localhost:3001",
		BuilderURL:      "http://localhost:3001",
		OrchestratorURL: "http://localhost:8090",
	}
}

// PromptOverwrite prompts user to confirm overwriting an existing profile
func PromptOverwrite(name, profile string) (bool, error) {
	fmt.Printf("Profile '%s/%s' already exists. Overwrite? [y/N]: ", name, profile)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}
