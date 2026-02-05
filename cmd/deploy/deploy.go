package deploy

import (
	"github.com/cozy-creator/cozyctl/internal/deploy"
	"github.com/spf13/cobra"
)

func DeployCmd() *cobra.Command {
	deployCmd := &cobra.Command{
		Use:   "deploy <build-id>",
		Short: "Deploy a build via cozy-hub",
		Long: `Deploy a previously built image using its build ID.

Cozy-hub will promote the build and register the deployment with the orchestrator.

This command will:
1. Read tenant-id from your config
2. Send build-id to cozy-hub
3. Cozy-hub promotes the build, registers with orchestrator

Example:
  cozyctl deploy abc-123-def-456`,
		Args: cobra.ExactArgs(1),
		RunE: runDeploy,
	}

	return deployCmd
}

func runDeploy(cmd *cobra.Command, args []string) error {
	buildID := args[0]
	return deploy.Run(buildID)
}
