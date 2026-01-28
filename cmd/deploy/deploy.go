package deploy

import (
	"github.com/cozy-creator/cozyctl/internal/deploy"
	"github.com/spf13/cobra"
)

var (
	flagRegister   bool
	flagDryRun     bool
	flagFunctions  string
	flagMinWorkers int
	flagMaxWorkers int
)

func DeployCmd() *cobra.Command {
	deployCmd := &cobra.Command{
		Use:   "deploy [path]",
		Short: "Build and register a new deployment",
		Long: `Build a Docker image from a Python project and register it as a new deployment.

The project must have a pyproject.toml with [tool.cozy] configuration.

This command will:
1. Parse pyproject.toml for deployment configuration
2. Generate a Dockerfile based on the configuration
3. Build the Docker image locally
4. Register the deployment with the orchestrator (if --register is true)

Example:
  cozyctl deploy .
  cozyctl deploy ./my-project
  cozyctl deploy ./my-project --dry-run
  cozyctl deploy ./my-project --register=false
  cozyctl deploy ./my-project --functions "generate:true,health:false"`,
		Args: cobra.MaximumNArgs(1),
		RunE: runDeploy,
	}

	deployCmd.Flags().BoolVar(&flagRegister, "register", true, "Register deployment with orchestrator after build")
	deployCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Show what would be done without executing")
	deployCmd.Flags().StringVar(&flagFunctions, "functions", "", "Comma-separated function specs (e.g., 'generate:true,health:false')")
	deployCmd.Flags().IntVar(&flagMinWorkers, "min-workers", 1, "Minimum number of workers")
	deployCmd.Flags().IntVar(&flagMaxWorkers, "max-workers", 0, "Maximum number of workers (0 = unlimited)")

	return deployCmd
}

func runDeploy(cmd *cobra.Command, args []string) error {
	projectPath := "."
	if len(args) > 0 {
		projectPath = args[0]
	}

	return deploy.Run(deploy.Options{
		ProjectPath: projectPath,
		Register:    flagRegister,
		DryRun:      flagDryRun,
		Functions:   flagFunctions,
		MinWorkers:  flagMinWorkers,
		MaxWorkers:  flagMaxWorkers,
	})
}
