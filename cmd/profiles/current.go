package profileCmd

import (
	"fmt"

	"github.com/cozy-creator/cozyctl/internal/config"
	"github.com/spf13/cobra"
)

// CurrentCmd shows the current profile
func CurrentCmd() *cobra.Command {
	currentCmd := &cobra.Command{
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

	return currentCmd
}
