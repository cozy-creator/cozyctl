package login

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"syscall"

	"github.com/cozy-creator/cozyctl/internal/config"
	"golang.org/x/term"
)

type TenantInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// RunLogin handles the login flow with name and profile
func RunLogin(apiKey, hubURL, builderURL, tenantID, name, profile string) error {
	// Get API key from various sources
	if apiKey == "" {
		apiKey = os.Getenv("COZY_API_KEY")
	}
	if apiKey == "" {
		var err error
		apiKey, err = PromptAPIKey()
		if err != nil {
			return err
		}
	}

	// Set defaults for name and profile
	if name == "" {
		name = "default"
	}
	if profile == "" {
		profile = "default"
	}

	// Check if profile already exists
	if config.ProfileExists(name, profile) {
		overwrite, err := config.PromptOverwrite(name, profile)
		if err != nil {
			return err
		}
		if !overwrite {
			return fmt.Errorf("login cancelled")
		}
	}

	fmt.Println("Authenticating...")

	// Validate the API key with cozy-hub
	tenant, err := ValidateAPIKey(hubURL, apiKey)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Use provided tenant ID or the one from validation
	if tenantID == "" {
		tenantID = tenant.ID
	}

	// Create profile config
	profileCfg := &config.ProfileConfig{
		CurrentName:    name,
		CurrentProfile: profile,
		Config: &config.ConfigData{
			HubURL:     hubURL,
			BuilderURL: builderURL,
			TenantID:   tenantID,
			Token:      apiKey,
		},
	}

	// Save profile config
	if err := config.SaveProfileConfig(name, profile, profileCfg); err != nil {
		return fmt.Errorf("failed to save profile config: %w", err)
	}

	// Update default pointer to this profile
	if err := config.SaveDefaultConfig(name, profile); err != nil {
		return fmt.Errorf("failed to save default config: %w", err)
	}

	configPath, _ := config.ProfileConfigPath(name, profile)
	fmt.Printf("Logged in as %s (tenant: %s)\n", tenant.Name, tenant.ID)
	fmt.Printf("Profile '%s/%s' saved to %s\n", name, profile, configPath)
	fmt.Printf("Set as current profile\n")

	return nil
}

// ImportConfig imports a config file into a profile
func ImportConfig(sourceFile, name, profile string) error {
	// Set defaults for name and profile
	if name == "" {
		name = "default"
	}
	if profile == "" {
		profile = "default"
	}

	// Check if profile already exists
	if config.ProfileExists(name, profile) {
		overwrite, err := config.PromptOverwrite(name, profile)
		if err != nil {
			return err
		}
		if !overwrite {
			return fmt.Errorf("import cancelled")
		}
	}

	// Import the config file
	profileCfg, err := config.ImportConfigFile(sourceFile, name, profile)
	if err != nil {
		return err
	}

	// Save the imported config
	if err := config.SaveProfileConfig(name, profile, profileCfg); err != nil {
		return fmt.Errorf("failed to save imported config: %w", err)
	}

	// Update default pointer to this profile
	if err := config.SaveDefaultConfig(name, profile); err != nil {
		return fmt.Errorf("failed to save default config: %w", err)
	}

	configPath, _ := config.ProfileConfigPath(name, profile)
	fmt.Printf("Config imported to profile '%s/%s' (%s)\n", name, profile, configPath)
	fmt.Printf("Set as current profile\n")

	return nil
}

func PromptAPIKey() (string, error) {
	fmt.Print("API Key: ")

	// Try to read password without echo
	if term.IsTerminal(int(syscall.Stdin)) {
		password, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println() // newline after hidden input
		if err != nil {
			return "", fmt.Errorf("failed to read API key: %w", err)
		}
		return strings.TrimSpace(string(password)), nil
	}

	// Fallback for non-terminal (piped input)
	reader := bufio.NewReader(os.Stdin)
	key, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read API key: %w", err)
	}
	return strings.TrimSpace(key), nil
}

func ValidateAPIKey(hubURL, apiKey string) (*TenantInfo, error) {
	url := strings.TrimRight(hubURL, "/") + "/api/v1/auth/me"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", hubURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("invalid API key")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var tenant TenantInfo
	if err := json.NewDecoder(resp.Body).Decode(&tenant); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &tenant, nil
}
