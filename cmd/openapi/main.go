package main

import (
	"fmt"
	"os"

	arazzoCmd "github.com/speakeasy-api/openapi/arazzo/cmd"
	openapiCmd "github.com/speakeasy-api/openapi/openapi/cmd"
	overlayCmd "github.com/speakeasy-api/openapi/overlay/cmd"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "openapi",
	Short: "OpenAPI toolkit for working with OpenAPI specifications, overlays, and Arazzo workflows",
	Long: `A comprehensive toolkit for working with OpenAPI specifications and Arazzo workflows.

This CLI provides tools for:

OpenAPI Specifications:
- Validate OpenAPI specification documents for compliance
- Upgrade OpenAPI specs to the latest supported version (3.1.1)
- Inline all references to create self-contained documents
- Bundle external references into components section while preserving structure

Arazzo Workflows:
- Validate Arazzo workflow documents for compliance

OpenAPI Overlays:
- Apply overlays to modify OpenAPI specifications
- Compare two specifications and generate overlays
- Validate overlay files for correctness

Each command group provides specialized functionality for working with different
aspects of the OpenAPI ecosystem, from basic validation to advanced document
transformation and workflow management.`,
	Version: version,
}

var overlayCmds = &cobra.Command{
	Use:   "overlay",
	Short: "Work with OpenAPI Overlays",
	Long: `Commands for working with OpenAPI Overlays.

OpenAPI Overlays provide a way to modify OpenAPI and Arazzo specifications
without directly editing the original files. This is useful for:
- Adding vendor-specific extensions
- Modifying specifications for different environments
- Applying transformations to third-party APIs`,
}

var openapiCmds = &cobra.Command{
	Use:   "openapi",
	Short: "Work with OpenAPI specifications",
	Long: `Commands for working with OpenAPI specifications.

OpenAPI specifications define REST APIs in a standard format.
These commands help you validate and work with OpenAPI documents.`,
}

var arazzoCmds = &cobra.Command{
	Use:   "arazzo",
	Short: "Work with Arazzo workflow documents",
	Long: `Commands for working with Arazzo workflow documents.

Arazzo workflows describe sequences of API calls and their dependencies.
These commands help you validate and work with Arazzo documents.`,
}

func init() {
	// Set version template
	rootCmd.SetVersionTemplate(`{{printf "%s" .Version}}`)

	// Add OpenAPI spec validation command
	openapiCmd.Apply(openapiCmds)

	// Add Arazzo workflow validation command
	arazzoCmd.Apply(arazzoCmds)

	// Add overlay subcommands using the Apply function
	overlayCmd.Apply(overlayCmds)

	// Add all commands to root
	rootCmd.AddCommand(openapiCmds)
	rootCmd.AddCommand(arazzoCmds)
	rootCmd.AddCommand(overlayCmds)

	// Global flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
