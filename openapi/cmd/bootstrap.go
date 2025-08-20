package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/yml"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap [output-file]",
	Short: "Create a new OpenAPI document with best practice examples",
	Long: `Bootstrap creates a new OpenAPI document template with comprehensive examples of best practices.

This command generates a complete OpenAPI specification that demonstrates:
• Proper document structure and metadata (info, servers, tags)
• Example operations with request/response definitions
• Reusable components (schemas, responses, security schemes)
• Reference usage ($ref) for component reuse
• Security scheme definitions (API key authentication)
• Comprehensive schema examples with validation rules

The generated document serves as both a template for new APIs and a learning
resource for OpenAPI best practices.

Examples:
  # Create bootstrap document and output to stdout
  openapi openapi bootstrap

  # Create bootstrap document and save to file
  openapi openapi bootstrap ./my-api.yaml

  # Create bootstrap document in current directory
  openapi openapi bootstrap ./openapi.yaml`,
	Args: cobra.MaximumNArgs(1),
	Run:  runBootstrap,
}

func runBootstrap(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	if err := createBootstrapDocument(ctx, args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func createBootstrapDocument(ctx context.Context, args []string) error {
	// Create the bootstrap document
	doc := openapi.Bootstrap()

	// Determine output destination
	var outputFile string
	writeToStdout := true

	if len(args) > 0 {
		outputFile = args[0]
		writeToStdout = false
	}

	// Create processor for output handling
	processor, err := NewOpenAPIProcessor("", outputFile, false)
	if err != nil {
		return err
	}

	// Override stdout setting based on our logic
	processor.WriteToStdout = writeToStdout

	// Write the document
	ctx = yml.ContextWithConfig(ctx, &yml.Config{
		ValueStringStyle: yaml.DoubleQuotedStyle,
		Indentation:      2,
		OutputFormat:     yml.OutputFormatYAML,
	})
	if err := processor.WriteDocument(ctx, doc); err != nil {
		return fmt.Errorf("failed to write bootstrap document: %w", err)
	}

	// Print success message (only if not writing to stdout)
	if !writeToStdout {
		processor.PrintSuccess("Bootstrap OpenAPI document created: " + outputFile)
	}

	return nil
}
