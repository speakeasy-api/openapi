package cmd

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
	Long: `Upgrade an OpenAPI specification document to the latest supported version (3.1.1).

This command will upgrade OpenAPI documents from:
- OpenAPI 3.0.x versions to 3.1.1 (always)
- OpenAPI 3.1.x versions to 3.1.1 (by default)
- Use --minor-only to only upgrade minor versions (3.0.x to 3.1.1, but skip 3.1.x versions)

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
	upgradeCmd.Flags().BoolVar(&minorOnly, "minor-only", false, "only upgrade minor versions (3.0.x to 3.1.1, skip 3.1.x versions)")
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

	if err := upgradeOpenAPI(ctx, processor, minorOnly); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func upgradeOpenAPI(ctx context.Context, processor *OpenAPIProcessor, minorOnly bool) error {
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
	if !minorOnly {
		// By default, upgrade all versions including patch versions (3.1.x to 3.1.1)
		opts = append(opts, openapi.WithUpgradeSamePatchVersion())
	}
	// When minorOnly is true, only 3.0.x versions will be upgraded to 3.1.1
	// 3.1.x versions will be skipped unless they need minor version upgrade

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
