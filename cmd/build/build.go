package build

import (
	"fmt"

	"github.com/cozy-creator/cozyctl/internal/build"
	"github.com/spf13/cobra"
)

var (
	BuildProjectDirectory string
	BuildProjectLocally   bool
)

func BuildCmd() *cobra.Command {
	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Build a project",
		Long: `Build a project on the Cozy platform.

By default, uploads the project to gen-builder for server-side building.
Use --local to build locally with Docker instead.

Examples:
  cozyctl build --dir ./my-project
  cozyctl build --local --dir ./my-project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if BuildProjectDirectory == "" {
				return fmt.Errorf("please specify a project path with --dir/-d")
			}
			if BuildProjectLocally {
				return build.BuildProjectLocally(BuildProjectDirectory)
			}
			return build.BuildProjectOnServer(BuildProjectDirectory)
		},
	}

	buildCmd.Flags().BoolVarP(&BuildProjectLocally, "local", "l", false, "Pass this if you want to build your project locally.")
	buildCmd.Flags().StringVarP(&BuildProjectDirectory, "dir", "d", "", "Pass in the project that you want to build.")

	return buildCmd
}
