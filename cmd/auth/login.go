package auth

import (
	authInternal "github.com/cozy-creator/cozyctl/internal/auth"
	"github.com/spf13/cobra"
)

var (
	loginAPIKey  string
	loginHubURL  string
	loginBuilder string
)

func LoginCmd(cfgFile *string) *cobra.Command {
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Cozy",
		Long: `Authenticate with the Cozy platform using your API key.

You can provide credentials via:
  1. Interactive prompt (default)
  2. Environment variables: COZY_API_KEY
  3. Flags: --api-key

Example:
  cozyctl auth login
  cozyctl auth login --api-key sk_live_xxx
  COZY_API_KEY=sk_live_xxx cozyctl auth login`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var cfgPath string
			if cfgFile != nil {
				cfgPath = *cfgFile
			}
			return authInternal.RunLogin(loginAPIKey, loginHubURL, loginBuilder, cfgPath)
		},
	}

	loginCmd.Flags().StringVar(&loginAPIKey, "api-key", "", "API key (or set COZY_API_KEY)")
	loginCmd.Flags().StringVar(&loginHubURL, "hub-url", "https://api.cozy.art", "Cozy Hub API URL")
	loginCmd.Flags().StringVar(&loginBuilder, "builder-url", "https://builder.cozy.art", "Gen-builder API URL")

	return loginCmd
}
