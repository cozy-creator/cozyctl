package auth

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"syscall"

	"github.com/cozy-creator/cozy-cli/internal/config"
	"golang.org/x/term"
)

type TenantInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func RunLogin(apiKey, hubURL, builderURL, cfgPath string) error {
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

	fmt.Println("Authenticating...")

	// Validate the API key with cozy-hub
	tenant, err := ValidateAPIKey(hubURL, apiKey)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Save config
	cfg := &config.Config{
		HubURL:     hubURL,
		BuilderURL: builderURL,
		TenantID:   tenant.ID,
		Token:      apiKey,
	}

	if err := config.Save(cfg, cfgPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	configPath, _ := config.ConfigPath()
	fmt.Printf("Logged in as %s (tenant: %s)\n", tenant.Name, tenant.ID)
	fmt.Printf("Config saved to %s\n", configPath)
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
