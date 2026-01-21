package profileCmd

import (
	"fmt"

	"github.com/cozy-creator/cozyctl/internal/config"
	"github.com/spf13/cobra"
)

func SwitchCmd() *cobra.Command {

	var useName string
	var useProfile string

	switchCmd := &cobra.Command{
		Use:   "use",
		Short: "Switch to a different profile",
		Long: `Switch the current name and/or profile.

You can switch both name and profile, or just one of them.

Examples:
  # Switch to a specific name and profile
  cozyctl use --name briheet --profile prod

  # Switch only the profile (keep current name)
  cozyctl use --profile staging

  # Switch only the name (keep current profile)
  cozyctl use --name damon`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Config
			defaultCfg, err := config.GetDefaultConfig()
			if err != nil {
				return err
			}

			// Determine new name and profile
			newName := useName
			if newName == "" {
				newName = defaultCfg.CurrentName
			}

			newProfile := useProfile
			if newProfile == "" {
				newProfile = defaultCfg.CurrentProfile
			}

			// Check if profile exists
			if !config.ProfileExists(newName, newProfile) {
				return fmt.Errorf("profile '%s/%s' does not exist", newName, newProfile)
			}

			// Save new default
			if err := config.SaveDefaultConfig(newName, newProfile); err != nil {
				return fmt.Errorf("failed to save default config: %w", err)
			}

			fmt.Printf("Switched to profile '%s/%s'\n", newName, newProfile)
			return nil
		},
	}

	switchCmd.Flags().StringVar(&useName, "name", "", "name to switch to")
	switchCmd.Flags().StringVar(&useProfile, "profile", "", "profile to switch to")

	return switchCmd
}
