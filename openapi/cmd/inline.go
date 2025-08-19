package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/spf13/cobra"
)

var inlineCmd = &cobra.Command{
	Use:   "inline <input-file> [output-file]",
	Short: "Inline all references in an OpenAPI specification",
	Long: `Inline all $ref references in an OpenAPI specification, creating a self-contained document.

This command transforms an OpenAPI document by replacing all $ref references with their actual content,
eliminating the need for external definitions or component references.

Benefits of inlining:
- Create standalone OpenAPI documents for easy distribution
- Improve compatibility with tools that work better with fully expanded specifications
- Provide complete, self-contained documents to AI systems and analysis tools
- Generate documentation where all schemas and components are visible inline
- Eliminate reference resolution overhead in performance-critical applications
- Debug API issues by seeing the full expanded document

The inlining process:
1. Resolves all component references (#/components/schemas/User, etc.)
2. Replaces $ref with the actual schema/response/parameter content
3. Handles circular references by using JSON Schema $defs
4. Optionally removes unused components after inlining

Output options:
- No output file specified: writes to stdout (pipe-friendly)
- Output file specified: writes to the specified file
- --write flag: writes in-place to the input file`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runInline,
}

var (
	inlineWriteInPlace bool
)

func init() {
	inlineCmd.Flags().BoolVarP(&inlineWriteInPlace, "write", "w", false, "write result in-place to input file")
}

func runInline(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	inputFile := args[0]

	var outputFile string
	if len(args) > 1 {
		outputFile = args[1]
	}

	processor, err := NewOpenAPIProcessor(inputFile, outputFile, inlineWriteInPlace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := inlineOpenAPI(ctx, processor); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func inlineOpenAPI(ctx context.Context, processor *OpenAPIProcessor) error {
	// Load the OpenAPI document
	doc, validationErrors, err := processor.LoadDocument(ctx)
	if err != nil {
		return err
	}
	if doc == nil {
		return errors.New("failed to parse OpenAPI document: document is nil")
	}

	// Report validation errors but continue with inlining
	processor.ReportValidationErrors(validationErrors)

	// Prepare inline options (always remove unused components)
	opts := openapi.InlineOptions{
		ResolveOptions: openapi.ResolveOptions{
			RootDocument:   doc,
			TargetLocation: filepath.Clean(processor.InputFile),
		},
		RemoveUnusedComponents: true,
	}

	// Perform the inlining
	if err := openapi.Inline(ctx, doc, opts); err != nil {
		return fmt.Errorf("failed to inline OpenAPI document: %w", err)
	}

	processor.PrintSuccess("Successfully inlined all references and removed unused components")

	return processor.WriteDocument(ctx, doc)
}

// GetInlineCommand returns the inline command for external use
func GetInlineCommand() *cobra.Command {
	return inlineCmd
}
