package build

import (
	"log"

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
		Short: "Manage builds",
		Long:  `Manage builds on the Cozy platform.`,
		RunE: func(cmd *cobra.Command, args []string) error {

			// Get current Configs
			// config, err := config.GetDefaultConfig()
			// if err != nil {
			// 	log.Fatal(err)
			// }

			// We would find the project by directory. Then check if local or not.
			// If local then build here else tar it up and send to the gen-builder to build.
			// This way it makes it compatible to build locally and github actions.
			if BuildProjectLocally {
				if BuildProjectDirectory == "" {
					log.Println("Please specifiy an path of the project you want to build.")
					return nil
				}
				err := build.BuildProjectLocally(BuildProjectDirectory)
				return err
			} else {
				// err := build.BuildProjectOnServer(config)
				return nil
			}
		},
	}

	buildCmd.Flags().BoolVarP(&BuildProjectLocally, "local", "l", false, "Pass this if you want to build your project locally.")
	buildCmd.Flags().StringVarP(&BuildProjectDirectory, "dir", "d", "", "Pass in the project that you want to build.")

	return buildCmd
}
