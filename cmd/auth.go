package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"syscall"

	"github.com/cozy-creator/cozy-cli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	loginAPIKey  string
	loginHubURL  string
	loginBuilder string
)

func AuthCmd() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Cozy",
		Long: `Authenticate with the Cozy platform using your API key.

You can provide credentials via:
  1. Interactive prompt (default)
  2. Environment variables: COZY_API_KEY
  3. Flags: --api-key

Example:
  cozy login
  cozy login --api-key sk_live_xxx
  COZY_API_KEY=sk_live_xxx cozy login`,
		RunE: runLogin,
	}

	authCmd.Flags().StringVar(&loginAPIKey, "api-key", "", "API key (or set COZY_API_KEY)")
	authCmd.Flags().StringVar(&loginHubURL, "hub-url", "https://api.cozy.art", "Cozy Hub API URL")
	authCmd.Flags().StringVar(&loginBuilder, "builder-url", "https://builder.cozy.art", "Gen-builder API URL")

	return authCmd
}

func runLogin(cmd *cobra.Command, args []string) error {
	apiKey := loginAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("COZY_API_KEY")
	}
	if apiKey == "" {
		var err error
		apiKey, err = promptAPIKey()
		if err != nil {
			return err
		}
	}

	fmt.Println("Authenticating...")

	// Validate the API key with cozy-hub
	tenant, err := validateAPIKey(loginHubURL, apiKey)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Save config
	cfg := &config.Config{
		HubURL:     loginHubURL,
		BuilderURL: loginBuilder,
		TenantID:   tenant.ID,
		Token:      apiKey,
	}

	if err := config.Save(cfg, cfgFile); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	configPath, _ := config.ConfigPath()
	fmt.Printf("Logged in as %s (tenant: %s)\n", tenant.Name, tenant.ID)
	fmt.Printf("Config saved to %s\n", configPath)
	return nil
}

func promptAPIKey() (string, error) {
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

type tenantInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func validateAPIKey(hubURL, apiKey string) (*tenantInfo, error) {
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

	var tenant tenantInfo
	if err := json.NewDecoder(resp.Body).Decode(&tenant); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &tenant, nil
}
