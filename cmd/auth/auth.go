package auth

import (
	"github.com/spf13/cobra"
)

func NewAuthCmd(cfgFile *string) *cobra.Command {
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
