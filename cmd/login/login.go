package login

import (
	authInternal "github.com/cozy-creator/cozyctl/internal/auth"
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
)

func LoginCmd() *cobra.Command {
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Cozy",
		Long: `Authenticate with the Cozy platform using your API key.

You can provide credentials via:
  1. Interactive prompt (default)
  2. Environment variables: COZY_API_KEY
  3. Flags: --api-key

Examples:
  # Login with default name and profile
  cozyctl login --api-key sk_live_xxx

  # Login with custom name and profile
  cozyctl login --name briheet --profile dev --api-key sk_live_xxx

  # Import existing config file
  cozyctl login --name briheet --profile prod --config-file ./prod-config.yaml

  # Using environment variable
  COZY_API_KEY=sk_live_xxx cozyctl login --name work --profile staging`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle config file import
			if loginConfigFile != "" {
				return authInternal.ImportConfig(loginConfigFile, loginName, loginProfile)
			}

			// Normal login flow
			return authInternal.RunLogin(
				loginAPIKey,
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
	loginCmd.Flags().StringVar(&loginAPIKey, "api-key", "", "API key (or set COZY_API_KEY)")
	loginCmd.Flags().StringVar(&loginConfigFile, "config-file", "", "import existing config file")
	loginCmd.Flags().StringVar(&loginHubURL, "hub-url", "https://api.cozy.art", "Cozy Hub API URL")
	loginCmd.Flags().StringVar(&loginBuilderURL, "builder-url", "https://builder.cozy.art", "Builder API URL")
	loginCmd.Flags().StringVar(&loginTenantID, "tenant-id", "", "tenant ID (usually auto-detected)")

	return loginCmd
}
