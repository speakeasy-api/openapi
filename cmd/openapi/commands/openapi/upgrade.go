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

By default, upgrades all versions including patch-level upgrades:
- 3.0.x → 3.2.0
- 3.1.x → 3.2.0
- 3.2.x (e.g., 3.2.0) → 3.2.0 (patch upgrade if newer patch exists)

With --minor-only, only performs cross-minor version upgrades:
- 3.0.x → 3.2.0 (cross-minor upgrade)
- 3.1.x → 3.2.0 (cross-minor upgrade)
- 3.2.x → no change (same minor version, skip patch upgrades)

The upgrade process includes:
- Updating the OpenAPI version field
- Converting nullable properties to proper JSON Schema format
- Updating schema validation rules
- Maintaining backward compatibility where possible

Output options:
- No output file specified: writes to stdout (pipe-friendly)
- Output file specified: writes to the specified file
- --write flag: writes in-place to the input file`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runUpgrade,
}

var (
	minorOnly    bool
	writeInPlace bool
)

func init() {
	upgradeCmd.Flags().BoolVar(&minorOnly, "minor-only", false, "only upgrade across minor versions, skip patch-level upgrades within same minor")
	upgradeCmd.Flags().BoolVarP(&writeInPlace, "write", "w", false, "write result in-place to input file")
}

func runUpgrade(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	inputFile := args[0]

	var outputFile string
	if len(args) > 1 {
		outputFile = args[1]
	}

	processor, err := NewOpenAPIProcessor(inputFile, outputFile, writeInPlace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := upgradeOpenAPI(ctx, processor, !minorOnly); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func upgradeOpenAPI(ctx context.Context, processor *OpenAPIProcessor, upgradeSameMinorVersion bool) error {
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

	// Prepare upgrade options
	var opts []openapi.Option[openapi.UpgradeOptions]
	if upgradeSameMinorVersion {
		// By default, upgrade all versions including patch upgrades (e.g., 3.2.0 → 3.2.1)
		opts = append(opts, openapi.WithUpgradeSameMinorVersion())
	}
	// When minorOnly is true, only cross-minor upgrades are performed
	// Patch upgrades within the same minor version (e.g., 3.2.0 → 3.2.1) are skipped

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
