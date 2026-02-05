package login

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"syscall"

	"github.com/cozy-creator/cozyctl/internal/config"
	"golang.org/x/term"
)

type TenantInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

type UserInfo struct {
	ID       string  `json:"id"`
	Username string  `json:"username"`
	Email    *string `json:"email"`
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
			HubURL:          hubURL,
			BuilderURL:      builderURL,
			OrchestratorURL: config.DefaultConfigData().OrchestratorURL,
			TenantID:        tenantID,
			Token:           apiKey,
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

// RunPasswordLogin handles the email/password login flow
func RunPasswordLogin(email, password, hubURL, builderURL, tenantID, name, profile string) error {
	// Get email/username from user
	if email == "" {
		var err error
		email, err = PromptEmail()
		if err != nil {
			return err
		}
	}

	// Validate identifier format (client-side validation)
	if err := ValidateIdentifier(email); err != nil {
		return fmt.Errorf("invalid email/username: %w", err)
	}

	// Get password
	if password == "" {
		var err error
		password, err = PromptPassword()
		if err != nil {
			return err
		}
	}

	// Validate password format (client-side validation)
	if err := ValidatePassword(password); err != nil {
		return fmt.Errorf("invalid password: %w", err)
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

	// Authenticate with AuthKit
	auth, err := PasswordLogin(hubURL, email, password)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Get user info to retrieve tenant ID
	userInfo, err := GetUserInfo(hubURL, auth.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	// Use provided tenant ID or the one from user info
	if tenantID == "" {
		tenantID = userInfo.ID
	}

	// Create profile config
	profileCfg := &config.ProfileConfig{
		CurrentName:    name,
		CurrentProfile: profile,
		Config: &config.ConfigData{
			HubURL:          hubURL,
			BuilderURL:      builderURL,
			OrchestratorURL: config.DefaultConfigData().OrchestratorURL,
			TenantID:        tenantID,
			Token:           auth.AccessToken,
			RefreshToken:    auth.RefreshToken,
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
	displayName := userInfo.Username
	if userInfo.Email != nil && *userInfo.Email != "" {
		displayName = *userInfo.Email
	}
	fmt.Printf("Logged in as %s (user: %s)\n", displayName, userInfo.ID)
	fmt.Printf("Profile '%s/%s' saved to %s\n", name, profile, configPath)
	fmt.Printf("Set as current profile\n")

	return nil
}

// PromptEmail prompts the user for their email or username
func PromptEmail() (string, error) {
	fmt.Print("Email or Username: ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read email: %w", err)
	}
	return strings.TrimSpace(input), nil
}

// PromptPassword prompts the user for their password (hidden input)
func PromptPassword() (string, error) {
	fmt.Print("Password: ")

	// Try to read password without echo
	if term.IsTerminal(int(syscall.Stdin)) {
		password, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println() // newline after hidden input
		if err != nil {
			return "", fmt.Errorf("failed to read password: %w", err)
		}
		return string(password), nil
	}

	// Fallback for non-terminal (piped input)
	reader := bufio.NewReader(os.Stdin)
	password, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	return strings.TrimSpace(password), nil
}

// ValidateIdentifier validates the login identifier (email, username, or phone)
func ValidateIdentifier(identifier string) error {
	identifier = strings.TrimSpace(identifier)

	if identifier == "" {
		return fmt.Errorf("email or username is required")
	}

	// If it contains @ it's an email - basic validation
	if strings.Contains(identifier, "@") {
		if len(identifier) < 5 { // minimum: a@b.c
			return fmt.Errorf("invalid email format")
		}
		return nil
	}

	// If it starts with + it's a phone number - basic validation
	if strings.HasPrefix(identifier, "+") {
		if len(identifier) < 10 {
			return fmt.Errorf("invalid phone number format")
		}
		return nil
	}

	// Username validation rules from AuthKit
	if len(identifier) < 4 {
		return fmt.Errorf("username must be at least 4 characters")
	}
	if len(identifier) > 30 {
		return fmt.Errorf("username must be at most 30 characters")
	}

	if len(identifier) > 0 {
		b0 := identifier[0]
		isLetter := (b0 >= 'a' && b0 <= 'z') || (b0 >= 'A' && b0 <= 'Z')
		if !isLetter {
			return fmt.Errorf("username must start with a letter")
		}
	}

	validPattern := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !validPattern.MatchString(identifier) {
		return fmt.Errorf("username can only contain letters, numbers, and underscores")
	}

	return nil
}

// ValidatePassword validates the password according to AuthKit rules
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	return nil
}

// PasswordLogin authenticates with AuthKit using username/password
func PasswordLogin(hubURL, login, password string) (*AuthResponse, error) {
	url := strings.TrimRight(hubURL, "/") + "/api/v1/auth/password/login"

	payload := map[string]string{
		"login":    login,
		"password": password,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", hubURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		// Try to get error message from response
		var errResp struct {
			Error string `json:"error"`
		}
		if json.NewDecoder(resp.Body).Decode(&errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("%s", errResp.Error)
		}
		return nil, fmt.Errorf("invalid credentials")
	}
	if resp.StatusCode != 200 {
		var errResp struct {
			Error string `json:"error"`
		}
		if json.NewDecoder(resp.Body).Decode(&errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("%s", errResp.Error)
		}
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var auth AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &auth, nil
}

// GetUserInfo retrieves user information using the access token
func GetUserInfo(hubURL, accessToken string) (*UserInfo, error) {
	url := strings.TrimRight(hubURL, "/") + "/api/v1/auth/user/me"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", hubURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var user UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &user, nil
}
