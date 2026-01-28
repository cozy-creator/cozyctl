package update

import (
	"github.com/cozy-creator/cozyctl/internal/update"
	"github.com/spf13/cobra"
)

var (
	flagDryRun     bool
	flagFunctions  string
	flagMinWorkers int
	flagMaxWorkers int
	flagImageOnly  bool
)

func UpdateCmd() *cobra.Command {
	updateCmd := &cobra.Command{
		Use:   "update [path]",
		Short: "Rebuild and update an existing deployment",
		Long: `Rebuild a Docker image and update an existing deployment.

The project must have a pyproject.toml with [tool.cozy] configuration.
The deployment must already exist (created with 'cozyctl deploy').

This command will:
1. Parse pyproject.toml for deployment configuration
2. Generate a Dockerfile based on the configuration
3. Build the Docker image locally
4. Update the existing deployment with the new image

Example:
  cozyctl update .
  cozyctl update ./my-project
  cozyctl update ./my-project --dry-run
  cozyctl update ./my-project --image-only
  cozyctl update ./my-project --functions "generate:true,health:false"`,
		Args: cobra.MaximumNArgs(1),
		RunE: runUpdate,
	}

	updateCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Show what would be done without executing")
	updateCmd.Flags().StringVar(&flagFunctions, "functions", "", "Comma-separated function specs (e.g., 'generate:true,health:false')")
	updateCmd.Flags().IntVar(&flagMinWorkers, "min-workers", -1, "Minimum number of workers (-1 = keep existing)")
	updateCmd.Flags().IntVar(&flagMaxWorkers, "max-workers", -1, "Maximum number of workers (-1 = keep existing)")
	updateCmd.Flags().BoolVar(&flagImageOnly, "image-only", false, "Only update the image, keep other settings")

	return updateCmd
}

func runUpdate(cmd *cobra.Command, args []string) error {
	projectPath := "."
	if len(args) > 0 {
		projectPath = args[0]
	}

	return update.Run(update.Options{
		ProjectPath: projectPath,
		DryRun:      flagDryRun,
		Functions:   flagFunctions,
		MinWorkers:  flagMinWorkers,
		MaxWorkers:  flagMaxWorkers,
		ImageOnly:   flagImageOnly,
	})
}
