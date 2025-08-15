package cmd

import "github.com/spf13/cobra"

var (
	rootCmd = &cobra.Command{
		Use:   "openapi-overlay",
		Short: "Work with OpenAPI Overlays",
	}
)

func init() {
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(compareCmd)
	rootCmd.AddCommand(validateCmd)
}

func Execute() {
	err := rootCmd.Execute()
	cobra.CheckErr(err)
}
