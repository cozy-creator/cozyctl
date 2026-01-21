package deploy

import (
	"github.com/spf13/cobra"
)

var (
	deployDeployment string
	deployPush       bool
	deployDryRun     bool
)

func DeployCmd() *cobra.Command {
	deployCmd := &cobra.Command{
		Use:   "deploy [path]",
		Short: "Deploy a project to Cozy",
		Long: `Deploy a Python project to the Cozy platform.

The project must have a pyproject.toml with [tool.cozy] configuration.

Example:
  cozy deploy .
  cozy deploy ./my-project
  cozy deploy --deployment my-model ./my-project`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return deployCmd
}
