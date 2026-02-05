package logout

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cozy-creator/cozyctl/internal/config"
)

// DefaultLogout clears the token for the current default profile
func DefaultLogout() error {
	// Get the current default profile
	defaultCfg, err := config.GetDefaultConfig()
	if err != nil {
		return fmt.Errorf("failed to get default config: %w", err)
	}

	// Clear the token for the default profile
	return clearProfileToken(defaultCfg.CurrentName, defaultCfg.CurrentProfile)
}

// NameOnlyLogout clears the tokens for all profiles under the given name
func NameOnlyLogout(name string) error {
	base, err := config.BaseDir()
	if err != nil {
		return err
	}

	nameDir := filepath.Join(base, name)

	// Check if name directory exists
	if _, err := os.Stat(nameDir); os.IsNotExist(err) {
		return fmt.Errorf("no profiles found for name '%s'", name)
	}

	// Read all profile directories under this name
	profileEntries, err := os.ReadDir(nameDir)
	if err != nil {
		return fmt.Errorf("failed to read profiles for name '%s': %w", name, err)
	}

	loggedOutCount := 0
	for _, entry := range profileEntries {
		if !entry.IsDir() {
			continue
		}

		// Check if config.yaml exists in this profile
		configPath := filepath.Join(nameDir, entry.Name(), "config.yaml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			continue
		}

		if err := clearProfileToken(name, entry.Name()); err != nil {
			fmt.Printf("Warning: failed to logout profile '%s/%s': %v\n", name, entry.Name(), err)
			continue
		}
		loggedOutCount++
	}

	if loggedOutCount == 0 {
		return fmt.Errorf("no profiles found for name '%s'", name)
	}

	fmt.Printf("Logged out of %d profile(s) under '%s'\n", loggedOutCount, name)
	return nil
}

// ProfileLogout clears the tokens for specific profiles under a name
func ProfileLogout(name string, profiles []string) error {
	loggedOutCount := 0
	for _, profile := range profiles {
		if !config.ProfileExists(name, profile) {
			fmt.Printf("Warning: profile '%s/%s' does not exist\n", name, profile)
			continue
		}

		if err := clearProfileToken(name, profile); err != nil {
			fmt.Printf("Warning: failed to logout profile '%s/%s': %v\n", name, profile, err)
			continue
		}
		loggedOutCount++
	}

	if loggedOutCount == 0 {
		return fmt.Errorf("no profiles were logged out")
	}

	fmt.Printf("Logged out of %d profile(s)\n", loggedOutCount)
	return nil
}

// clearProfileToken clears the token and refresh token for a specific profile
func clearProfileToken(name, profile string) error {
	// Get the profile config
	profileCfg, err := config.GetProfileConfig(name, profile)
	if err != nil {
		return err
	}

	// Check if already logged out
	if profileCfg.Config == nil || profileCfg.Config.Token == "" {
		fmt.Printf("Profile '%s/%s' is already logged out\n", name, profile)
		return nil
	}

	// Clear the tokens
	profileCfg.Config.Token = ""
	profileCfg.Config.RefreshToken = ""

	// Save the updated config
	if err := config.SaveProfileConfig(name, profile, profileCfg); err != nil {
		return fmt.Errorf("failed to save profile config: %w", err)
	}

	fmt.Printf("Logged out of profile '%s/%s'\n", name, profile)
	return nil
}
