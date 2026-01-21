package authCmd

import (
	"github.com/spf13/cobra"
)

func AuthCmd(cfgFile *string) *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
		Long: `Manage authentication with the Cozy platform.

Subcommands:
  login   Authenticate with Cozy`,
	}

	authCmd.AddCommand(LoginCmd(cfgFile))

	return authCmd
}
