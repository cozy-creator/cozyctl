package cmd

import (
	"fmt"
	"slices"

	"github.com/cozy-creator/cozyctl/cmd/build"
	"github.com/cozy-creator/cozyctl/cmd/deploy"
	"github.com/cozy-creator/cozyctl/cmd/login"
	logoutCmd "github.com/cozy-creator/cozyctl/cmd/logout"
	profileCmd "github.com/cozy-creator/cozyctl/cmd/profiles"
	"github.com/cozy-creator/cozyctl/internal/config"
	"github.com/spf13/cobra"
)

var (
	nameFlag    string
	profileFlag string
	profileCfg  *config.ProfileConfig
)

func Execute() error {
	var rootCmd = &cobra.Command{
		Use:   "cozyctl",
		Short: "cozyctl - deploy and manage ML functions",
		Long: `cozyctl is a command-line tool for deploying and managing
machine learning functions on the Cozy platform.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Skip config loading for these commands
			skipCommands := []string{"login", "profiles", "use", "current", "delete"}
			isTrue := slices.Contains(skipCommands, cmd.Name())
			if isTrue {
				return nil
			}

			// Get default config (pointer to current name+profile)
			defaultCfg, err := config.GetDefaultConfig()
			if err != nil {
				return fmt.Errorf("failed to load default config: %w", err)
			}

			// Determine which name and profile to use
			name := nameFlag
			if name == "" {
				name = defaultCfg.CurrentName
			}

			profile := profileFlag
			if profile == "" {
				profile = defaultCfg.CurrentProfile
			}

			// Load the profile config
			profileCfg, err = config.GetProfileConfig(name, profile)
			if err != nil {
				return err
			}

			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&nameFlag, "name", "", "name to use for this command")
	rootCmd.PersistentFlags().StringVar(&profileFlag, "profile", "", "profile to use for this command")

	rootCmd.AddCommand(loginCmd.LoginCmd())
	rootCmd.AddCommand(logoutCmd.LogoutCmd())
	rootCmd.AddCommand(deploy.DeployCmd())
	rootCmd.AddCommand(build.BuildCmd())
	rootCmd.AddCommand(profileCmd.ProfileCmd())

	return rootCmd.Execute()
}
