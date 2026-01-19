package overlay

import "github.com/spf13/cobra"

func Apply(rootCmd *cobra.Command) {
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(compareCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(validateCmd)
}
