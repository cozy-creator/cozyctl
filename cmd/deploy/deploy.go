package deploy

import (
	"github.com/cozy-creator/cozyctl/internal/deploy"
	"github.com/spf13/cobra"
)

func DeployCmd() *cobra.Command {
	deployCmd := &cobra.Command{
		Use:   "deploy <build-id>",
		Short: "Deploy a build to the orchestrator",
		Long: `Deploy a previously built image using its build ID.

The orchestrator will fetch the build metadata from S3 and register the deployment.

This command will:
1. Read tenant-id from your config
2. Send build-id and tenant-id to the orchestrator
3. The orchestrator handles fetching metadata from S3 and deploying

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
