package profiles

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/cozy-creator/cozyctl/internal/config"
	"github.com/spf13/cobra"
)

// ProfilesCmd lists all profiles
func ProfilesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "profiles",
		Short: "List all profiles",
		Long: `List all configured name/profile combinations.

The currently active profile is marked with an asterisk (*).

Example:
  cozyctl profiles`,
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := config.ListAllProfiles()
			if err != nil {
				return err
			}

			if len(profiles) == 0 {
				fmt.Println("No profiles found. Run 'cozyctl login' to create one.")
				return nil
			}

			// Sort profiles by name, then by profile
			sort.Slice(profiles, func(i, j int) bool {
				if profiles[i].Name != profiles[j].Name {
					return profiles[i].Name < profiles[j].Name
				}
				return profiles[i].Profile < profiles[j].Profile
			})

			// Get current profile
			defaultCfg, err := config.GetDefaultConfig()
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tPROFILE\tCURRENT")
			for _, p := range profiles {
				marker := ""
				if p.Name == defaultCfg.CurrentName && p.Profile == defaultCfg.CurrentProfile {
					marker = "*"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name, p.Profile, marker)
			}
			w.Flush()

			return nil
		},
	}
}

// CurrentCmd shows the current profile
func CurrentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show current profile",
		Long: `Display the currently active name/profile combination.

Example:
  cozyctl current`,
		RunE: func(cmd *cobra.Command, args []string) error {
			defaultCfg, err := config.GetDefaultConfig()
			if err != nil {
				return err
			}

			fmt.Printf("%s/%s\n", defaultCfg.CurrentName, defaultCfg.CurrentProfile)
			return nil
		},
	}
}

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
