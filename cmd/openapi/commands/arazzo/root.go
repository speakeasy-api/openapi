package arazzo

import "github.com/spf13/cobra"

// Apply adds Arazzo commands to the provided root command
func Apply(rootCmd *cobra.Command) {
	rootCmd.AddCommand(validateCmd)
}
