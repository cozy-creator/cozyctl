package profileCmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/cozy-creator/cozyctl/internal/config"
	"github.com/spf13/cobra"
)

// ProfilesCmd lists all profiles
func ProfileCmd() *cobra.Command {
	profileCmd := &cobra.Command{
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

	profileCmd.AddCommand(SwitchCmd())
	profileCmd.AddCommand(CurrentCmd())
	profileCmd.AddCommand(DeleteCmd())

	return profileCmd
}
