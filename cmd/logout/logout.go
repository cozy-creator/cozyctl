package logoutCmd

import (
	"github.com/cozy-creator/cozyctl/internal/logout"
	"github.com/spf13/cobra"
)

var (
	name    string
	profile []string
)

func LogoutCmd() *cobra.Command {
	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout of the system",
		Long: `Logout of the Cozy platform.

Examples:
  # Logout with name. This will logout all the profiles.
  cozyctl logout --name <put-your-name-here>

  # Logout with name and a profile/profiles. It can be one, can be many profiles.
  cozyctl logout --name <put-your-name-here> --profile <put-your-profile-here> <put-your-profile-here>
`,
		RunE: func(cmd *cobra.Command, args []string) error {

			if name == "" {
				// This means the person wants to logout the current default
				// Get the default and clear the token in the config. Writes persist to disk hence do it.
				return logout.DefaultLogout()
			}

			if len(profile) == 0 {
				// This means get all the configs in the directory of the name we have got and clear the token.
				return logout.NameOnlyLogout()
			}

			return nil
		},
	}

	logoutCmd.Flags().StringVar(&name, "name", "", "name/account identifier (default: 'default')")
	logoutCmd.Flags().StringSliceVar(&profile, "profile", []string{""}, "profile/environment (default: 'default')")

	return logoutCmd
}
