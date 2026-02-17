package openapi

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade <input-file> [output-file]",
	Short: "Upgrade an OpenAPI specification to the latest supported version",
	Long: `Upgrade an OpenAPI specification document to the latest supported version (3.2.0).

Stdin is supported — either pipe data directly or use '-' explicitly:
  cat spec.yaml | openapi spec upgrade
  cat spec.yaml | openapi spec upgrade -

By default, upgrades all versions including patch-level upgrades:
- 3.0.x → 3.2.0
- 3.1.x → 3.2.0
- 3.2.x (e.g., 3.2.0) → 3.2.0 (patch upgrade if newer patch exists)

With --minor-only, only performs cross-minor version upgrades:
- 3.0.x → 3.2.0 (cross-minor upgrade)
- 3.1.x → 3.2.0 (cross-minor upgrade)
- 3.2.x → no change (same minor version, skip patch upgrades)

With --version, upgrades to a specific OpenAPI version instead of the latest:
- openapi spec upgrade --version 3.1.0 spec.yaml
  (upgrades a 3.0.x spec to 3.1.0)

Note: --version and --minor-only are mutually exclusive.

The upgrade process includes:
- Updating the OpenAPI version field
- Converting nullable properties to proper JSON Schema format
- Updating schema validation rules
- Maintaining backward compatibility where possible

Output options:
- No output file specified: writes to stdout (pipe-friendly)
- Output file specified: writes to the specified file
- --write flag: writes in-place to the input file`,
	Args: stdinOrFileArgs(1, 2),
	Run:  runUpgrade,
}

var (
	minorOnly     bool
	writeInPlace  bool
	targetVersion string
)

func init() {
	upgradeCmd.Flags().BoolVar(&minorOnly, "minor-only", false, "only upgrade across minor versions, skip patch-level upgrades within same minor")
	upgradeCmd.Flags().BoolVarP(&writeInPlace, "write", "w", false, "write result in-place to input file")
	upgradeCmd.Flags().StringVarP(&targetVersion, "version", "V", "", "target OpenAPI version to upgrade to (default latest)")
}

func runUpgrade(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	inputFile := inputFileFromArgs(args)

	if targetVersion != "" && minorOnly {
		fmt.Fprintf(os.Stderr, "Error: --version and --minor-only are mutually exclusive\n")
		os.Exit(1)
	}

	outputFile := outputFileFromArgs(args)

	processor, err := NewOpenAPIProcessor(inputFile, outputFile, writeInPlace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var opts []openapi.Option[openapi.UpgradeOptions]
	if targetVersion != "" {
		opts = append(opts, openapi.WithUpgradeTargetVersion(targetVersion))
		// Enable same-minor upgrades so patch-level targets work as expected.
		// Without this, --version 3.1.2 on a 3.1.0 doc would be silently
		// skipped because they share the same minor version.
		opts = append(opts, openapi.WithUpgradeSameMinorVersion())
	} else if !minorOnly {
		// By default, upgrade all versions including patch upgrades (e.g., 3.2.0 → 3.2.1)
		opts = append(opts, openapi.WithUpgradeSameMinorVersion())
	}
	// When minorOnly is true, only cross-minor upgrades are performed
	// Patch upgrades within the same minor version (e.g., 3.2.0 → 3.2.1) are skipped

	if err := upgradeOpenAPI(ctx, processor, opts...); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func upgradeOpenAPI(ctx context.Context, processor *OpenAPIProcessor, opts ...openapi.Option[openapi.UpgradeOptions]) error {
	// Load the OpenAPI document
	doc, validationErrors, err := processor.LoadDocument(ctx)
	if err != nil {
		return err
	}
	if doc == nil {
		return errors.New("failed to parse OpenAPI document: document is nil")
	}

	// Report validation errors but continue with upgrade
	processor.ReportValidationErrors(validationErrors)

	// Perform the upgrade
	originalVersion := doc.OpenAPI
	upgraded, err := openapi.Upgrade(ctx, doc, opts...)
	if err != nil {
		return fmt.Errorf("failed to upgrade OpenAPI document: %w", err)
	}

	if !upgraded {
		processor.PrintInfo("No upgrade needed - document is already at version " + originalVersion)
		// Still output the document even if no upgrade was needed
		return processor.WriteDocument(ctx, doc)
	}

	processor.PrintSuccess(fmt.Sprintf("Successfully upgraded from %s to %s", originalVersion, doc.OpenAPI))

	return processor.WriteDocument(ctx, doc)
}

// GetUpgradeCommand returns the upgrade command for external use
func GetUpgradeCommand() *cobra.Command {
	return upgradeCmd
}
