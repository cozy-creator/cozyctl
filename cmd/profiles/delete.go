package profileCmd

import (
	"fmt"

	"github.com/cozy-creator/cozyctl/internal/config"
	"github.com/spf13/cobra"
)

// DeleteCmd deletes a profile
func DeleteCmd() *cobra.Command {
	var deleteName string
	var deleteProfile string

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a profile",
		Long: `Delete a specific name/profile configuration.

Note: Cannot delete the default/default profile.
If you delete the currently active profile, it will automatically switch to default/default.

Example:
  cozyctl delete --name briheet --profile staging`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Both name and profile are required
			if deleteName == "" || deleteProfile == "" {
				return fmt.Errorf("both --name and --profile flags are required")
			}

			// Delete the profile
			if err := config.DeleteProfile(deleteName, deleteProfile); err != nil {
				return err
			}

			fmt.Printf("Profile '%s/%s' deleted\n", deleteName, deleteProfile)

			// Check if we deleted the current profile
			defaultCfg, err := config.GetDefaultConfig()
			if err != nil {
				return err
			}

			if defaultCfg.CurrentName == deleteName && defaultCfg.CurrentProfile == deleteProfile {
				// Switch to default/default
				if err := config.SaveDefaultConfig("default", "default"); err != nil {
					return fmt.Errorf("failed to switch to default profile: %w", err)
				}
				fmt.Println("Switched to default/default profile")
			}

			return nil
		},
	}

	deleteCmd.Flags().StringVar(&deleteName, "name", "", "name to delete (required)")
	deleteCmd.Flags().StringVar(&deleteProfile, "profile", "", "profile to delete (required)")
	deleteCmd.MarkFlagRequired("name")
	deleteCmd.MarkFlagRequired("profile")

	return deleteCmd
}
