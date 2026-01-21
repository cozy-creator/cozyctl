package cmd

import (
	authCmd "github.com/cozy-creator/cozyctl/cmd/auth"
	"github.com/cozy-creator/cozyctl/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
)

func Execute() error {

	var rootCmd = &cobra.Command{
		Use:   "cozyctl",
		Short: "cozyctl - deploy and manage ML functions",
		Long: `cozyctl is a command-line tool for deploying and managing
machine learning functions on the Cozy platform.

Commands:
  auth      Helps in login, logout, whoami commands
  deploy    Deploy a project to Cozy
  builds    Manage builds (list, logs, cancel)`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Skip config loading for login command
			if cmd.Name() == "login" {
				return nil
			}

			var err error
			cfg, err = config.Load(cfgFile)
			if err != nil {
				// Config is optional for some commands
				cfg = config.Default()
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ~/.cozy/config.yaml)")

	rootCmd.AddCommand(authCmd.AuthCmd(&cfgFile))
	rootCmd.AddCommand(DeployCmd())
	rootCmd.AddCommand(BuildsCmd())

	return rootCmd.Execute()
}
