package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	arazzoCmd "github.com/speakeasy-api/openapi/cmd/openapi/commands/arazzo"
	openapiCmd "github.com/speakeasy-api/openapi/cmd/openapi/commands/openapi"
	overlayCmd "github.com/speakeasy-api/openapi/cmd/openapi/commands/overlay"
	swaggerCmd "github.com/speakeasy-api/openapi/cmd/openapi/commands/swagger"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// getVersionInfo returns version information, prioritizing ldflags values over build info
func getVersionInfo() (string, string, string) {
	// If version/commit/date were set via ldflags (GoReleaser), use those
	if version != "dev" || commit != "none" || date != "unknown" {
		return version, commit, date
	}

	// Otherwise, try to get info from build info
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return version, commit, date
	}

	// Use module version if available, otherwise fallback to "dev"
	moduleVersion := version
	if buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
		moduleVersion = buildInfo.Main.Version
	}

	// Extract VCS information
	vcsCommit := commit
	vcsTime := date

	for _, setting := range buildInfo.Settings {
		switch setting.Key {
		case "vcs.revision":
			if len(setting.Value) >= 7 {
				vcsCommit = setting.Value[:7] // Short commit hash
			} else {
				vcsCommit = setting.Value
			}
		case "vcs.time":
			vcsTime = setting.Value
		}
	}

	return moduleVersion, vcsCommit, vcsTime
}

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
	Use:   "spec",
	Short: "Work with OpenAPI specifications",
	Long: `Commands for working with OpenAPI specifications.

OpenAPI specifications define REST APIs in a standard format.
These commands help you validate and work with OpenAPI documents.`,
}

var swaggerCmds = &cobra.Command{
	Use:   "swagger",
	Short: "Work with Swagger 2.0 (OpenAPI v2) specifications",
	Long: `Commands for working with Swagger 2.0 (OpenAPI v2) specifications.

Swagger 2.0 documents describe REST APIs prior to OpenAPI 3.x.
These commands help you validate and upgrade Swagger documents.`,
}

var arazzoCmds = &cobra.Command{
	Use:   "arazzo",
	Short: "Work with Arazzo workflow documents",
	Long: `Commands for working with Arazzo workflow documents.

Arazzo workflows describe sequences of API calls and their dependencies.
These commands help you validate and work with Arazzo documents.`,
}

func init() {
	// Get version information (prioritizes ldflags, falls back to build info)
	currentVersion, currentCommit, currentDate := getVersionInfo()

	// Update root command version
	rootCmd.Version = currentVersion

	// Set version template with build info
	var versionTemplate strings.Builder
	versionTemplate.WriteString(`{{printf "%s" .Version}}`)

	if currentCommit != "none" && currentCommit != "" {
		versionTemplate.WriteString("\nBuild: " + currentCommit)
	}

	if currentDate != "unknown" && currentDate != "" {
		versionTemplate.WriteString("\nBuilt: " + currentDate)
	}

	rootCmd.SetVersionTemplate(versionTemplate.String())

	// Add OpenAPI spec validation command
	openapiCmd.Apply(openapiCmds)

	// Add Swagger 2.0 commands
	swaggerCmd.Apply(swaggerCmds)

	// Add Arazzo workflow validation command
	arazzoCmd.Apply(arazzoCmds)

	// Add overlay subcommands using the Apply function
	overlayCmd.Apply(overlayCmds)

	// Add all commands to root
	rootCmd.AddCommand(openapiCmds)
	rootCmd.AddCommand(swaggerCmds)
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
