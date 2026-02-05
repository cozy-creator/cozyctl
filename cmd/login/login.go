package loginCmd

import (
	"os"

	"github.com/cozy-creator/cozyctl/internal/login"
	"github.com/spf13/cobra"
)

var (
	loginAPIKey     string
	loginHubURL     string
	loginBuilderURL string
	loginTenantID   string
	loginName       string
	loginProfile    string
	loginConfigFile string
	loginEmail      string
	loginPassword   string
)

func LoginCmd() *cobra.Command {
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Cozy",
		Long: `Authenticate with the Cozy platform using email/password or API key.

You can provide credentials via:
  1. Interactive prompt (default - prompts for email and password)
  2. Flags: --email and --password
  3. API key: --api-key or COZY_API_KEY environment variable

Examples:
  # Interactive login (prompts for email and password)
  cozyctl login

  # Login with email and password
  cozyctl login --email user@example.com --password mypass

  # Login with username instead of email
  cozyctl login --email myusername --password mypass

  # Login with custom profile
  cozyctl login --name briheet --profile dev

  # Login with API key
  cozyctl login --api-key sk_live_xxx

  # Import existing config file
  cozyctl login --name briheet --profile prod --config-file ./prod-config.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle config file import
			if loginConfigFile != "" {
				return login.ImportConfig(loginConfigFile, loginName, loginProfile)
			}

			// Check for API key from flag or environment
			apiKey := loginAPIKey
			if apiKey == "" {
				apiKey = os.Getenv("COZY_API_KEY")
			}

			// If API key is provided, use the API key flow
			if apiKey != "" {
				return login.RunLogin(
					apiKey,
					loginHubURL,
					loginBuilderURL,
					loginTenantID,
					loginName,
					loginProfile,
				)
			}

			// Email/password login flow
			return login.RunPasswordLogin(
				loginEmail,
				loginPassword,
				loginHubURL,
				loginBuilderURL,
				loginTenantID,
				loginName,
				loginProfile,
			)
		},
	}

	loginCmd.Flags().StringVar(&loginName, "name", "", "name/account identifier (default: 'default')")
	loginCmd.Flags().StringVar(&loginProfile, "profile", "", "profile/environment (default: 'default')")
	loginCmd.Flags().StringVarP(&loginEmail, "email", "e", "", "email or username for login")
	loginCmd.Flags().StringVarP(&loginPassword, "password", "p", "", "password for login")
	loginCmd.Flags().StringVar(&loginAPIKey, "api-key", "", "API key (or set COZY_API_KEY)")
	loginCmd.Flags().StringVar(&loginConfigFile, "config-file", "", "import existing config file")
	loginCmd.Flags().StringVar(&loginHubURL, "hub-url", "https://api.cozy.art", "Cozy Hub API URL")
	loginCmd.Flags().StringVar(&loginBuilderURL, "builder-url", "https://api.cozy.art", "Builder API URL (now part of cozy-hub)")
	loginCmd.Flags().StringVar(&loginTenantID, "tenant-id", "", "tenant ID (usually auto-detected)")

	return loginCmd
}
