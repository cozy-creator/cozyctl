package build

import (
	"github.com/cozy-creator/cozyctl/internal/config"
	"github.com/spf13/cobra"
)

func BuildCmd(getConfig func() *config.ProfileConfig) *cobra.Command {
	buildsCmd := &cobra.Command{
		Use:   "builds",
		Short: "Manage builds",
		Long: `Manage builds on the Cozy platform.

Subcommands:
  list    List recent builds
  logs    View build logs
  cancel  Cancel a running build`,
	}

	return buildsCmd
}
