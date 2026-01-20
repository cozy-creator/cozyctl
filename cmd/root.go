package cmd

import (
	"fmt"
	"os"

	"github.com/cozy-creator/cozy-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "cozy",
	Short: "Cozy CLI - deploy and manage ML functions",
	Long: `Cozy CLI is a command-line tool for deploying and managing
machine learning functions on the Cozy platform.

Commands:
  login     Authenticate with Cozy
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

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ~/.cozy/config.yaml)")

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(buildsCmd)
}

func exitError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}
