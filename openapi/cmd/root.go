package cmd

import "github.com/spf13/cobra"

// Apply adds OpenAPI commands to the provided root command
func Apply(rootCmd *cobra.Command) {
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(inlineCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(bundleCmd)
	rootCmd.AddCommand(joinCmd)
	rootCmd.AddCommand(bootstrapCmd)
	rootCmd.AddCommand(optimizeCmd)
	rootCmd.AddCommand(localizeCmd)
}
