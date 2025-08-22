package cmd

import (
	"context"
	"fmt"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/spf13/cobra"
)

// bundleWriteInPlace controls whether to write the bundled document back to the input file
var bundleWriteInPlace bool

// bundleNamingStrategy controls the naming strategy for bundled components
var bundleNamingStrategy string

var bundleCmd = &cobra.Command{
	Use:   "bundle [input-file] [output-file]",
	Short: "Bundle external references into components section",
	Long: `Bundle transforms an OpenAPI document by bringing all external references into the components section,
creating a self-contained document that maintains the reference structure but doesn't depend on external files.

This operation is useful when you want to:
• Create portable documents that combine multiple OpenAPI files
• Maintain reference structure for tooling that supports references
• Simplify distribution by sharing a single file with all dependencies
• Prepare documents for further processing or transformations

The bundle command supports two naming strategies:
• counter: Uses counter-based suffixes like User_1, User_2 for conflicts
• filepath: Uses file path-based naming like external_api_yaml~User

Examples:
  # Bundle to stdout (pipe-friendly)
  openapi spec bundle ./spec-with-refs.yaml

  # Bundle to specific file
  openapi spec bundle ./spec.yaml ./bundled-spec.yaml

  # Bundle in-place with counter naming
  openapi spec bundle -w --naming counter ./spec.yaml

  # Bundle with filepath naming (default)
  openapi spec bundle --naming filepath ./spec.yaml ./bundled.yaml`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runBundleCommand,
}

func init() {
	bundleCmd.Flags().BoolVarP(&bundleWriteInPlace, "write", "w", false, "Write bundled document back to input file")
	bundleCmd.Flags().StringVar(&bundleNamingStrategy, "naming", "filepath", "Naming strategy for bundled components (counter|filepath)")
}

func runBundleCommand(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Parse arguments
	inputFile := args[0]
	var outputFile string
	if len(args) > 1 {
		outputFile = args[1]
	}

	// Validate naming strategy
	var namingStrategy openapi.BundleNamingStrategy
	switch bundleNamingStrategy {
	case "counter":
		namingStrategy = openapi.BundleNamingCounter
	case "filepath":
		namingStrategy = openapi.BundleNamingFilePath
	default:
		return fmt.Errorf("invalid naming strategy: %s (must be 'counter' or 'filepath')", bundleNamingStrategy)
	}

	// Create processor
	processor, err := NewOpenAPIProcessor(inputFile, outputFile, bundleWriteInPlace)
	if err != nil {
		return err
	}

	// Load document
	doc, validationErrors, err := processor.LoadDocument(ctx)
	if err != nil {
		return err
	}

	// Report validation errors (if any)
	processor.ReportValidationErrors(validationErrors)

	// Configure bundle options
	opts := openapi.BundleOptions{
		ResolveOptions: openapi.ResolveOptions{
			RootDocument:   doc,
			TargetLocation: inputFile,
		},
		NamingStrategy: namingStrategy,
	}

	// Bundle the document
	if err := openapi.Bundle(ctx, doc, opts); err != nil {
		return fmt.Errorf("failed to bundle document: %w", err)
	}

	// Print success message
	processor.PrintSuccess("Successfully bundled all external references into components section")

	// Write the bundled document
	if err := processor.WriteDocument(ctx, doc); err != nil {
		return err
	}

	return nil
}
