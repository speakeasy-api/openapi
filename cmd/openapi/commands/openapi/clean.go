package openapi

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean <input-file> [output-file]",
	Short: "Remove unused components and unused top-level tags from an OpenAPI specification",
	Long: `Remove unused components and unused top-level tags from an OpenAPI specification to create a cleaner, more focused document.

Use '-' as the input file to read from stdin:
  cat spec.yaml | openapi spec clean -

This command uses reachability-based analysis to keep only what is actually used by the API surface:
- Seeds reachability exclusively from API surface areas: entries under /paths and the top-level security section
- Expands through $ref links across component sections until a fixed point is reached
- Preserves security schemes referenced by name in security requirement objects (global or operation-level)
- Prunes any components that are not reachable from the API surface
- Removes unused top-level tags that are not referenced by any operation

What gets cleaned:
- Unused schemas in components/schemas
- Unused responses in components/responses
- Unused parameters in components/parameters
- Unused examples in components/examples
- Unused request bodies in components/requestBodies
- Unused headers in components/headers
- Unused security schemes in components/securitySchemes (with special handling)
- Unused links in components/links
- Unused callbacks in components/callbacks
- Unused path items in components/pathItems
- Unused top-level tags (global tags not referenced by any operation)

Special handling for security schemes:
Security schemes can be referenced in two ways:
1. By $ref (like other components)
2. By name in security requirement objects (global or operation-level)
The clean command correctly handles both cases and preserves security schemes that are referenced by name in security blocks.

Benefits of cleaning:
- Reduce document size by removing dead code
- Improve clarity by keeping only used components
- Optimize tooling performance with smaller documents
- Maintain clean specifications for distribution
- Prepare documents for sharing or publishing

Output options:
- No output file specified: writes to stdout (pipe-friendly)
- Output file specified: writes to the specified file
- --write flag: writes in-place to the input file`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runClean,
}

var cleanWriteInPlace bool

func init() {
	cleanCmd.Flags().BoolVarP(&cleanWriteInPlace, "write", "w", false, "write result in-place to input file")
}

func runClean(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	inputFile := args[0]

	var outputFile string
	if len(args) > 1 {
		outputFile = args[1]
	}

	processor, err := NewOpenAPIProcessor(inputFile, outputFile, cleanWriteInPlace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := cleanOpenAPI(ctx, processor); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func cleanOpenAPI(ctx context.Context, processor *OpenAPIProcessor) error {
	// Load the OpenAPI document
	doc, validationErrors, err := processor.LoadDocument(ctx)
	if err != nil {
		return err
	}
	if doc == nil {
		return errors.New("failed to parse OpenAPI document: document is nil")
	}

	// Report validation errors but continue with cleaning
	processor.ReportValidationErrors(validationErrors)

	// Count components before cleaning
	componentCounts := countComponents(doc)

	// Perform the cleaning
	if err := openapi.Clean(ctx, doc); err != nil {
		return fmt.Errorf("failed to clean OpenAPI document: %w", err)
	}

	// Count components after cleaning and report
	newComponentCounts := countComponents(doc)
	reportCleaningResults(processor, componentCounts, newComponentCounts)

	return processor.WriteDocument(ctx, doc)
}

// countComponents counts the number of components in each section
func countComponents(doc *openapi.OpenAPI) map[string]int {
	counts := make(map[string]int)

	if doc.Components == nil {
		return counts
	}

	if doc.Components.Schemas != nil {
		counts["schemas"] = doc.Components.Schemas.Len()
	}
	if doc.Components.Responses != nil {
		counts["responses"] = doc.Components.Responses.Len()
	}
	if doc.Components.Parameters != nil {
		counts["parameters"] = doc.Components.Parameters.Len()
	}
	if doc.Components.Examples != nil {
		counts["examples"] = doc.Components.Examples.Len()
	}
	if doc.Components.RequestBodies != nil {
		counts["requestBodies"] = doc.Components.RequestBodies.Len()
	}
	if doc.Components.Headers != nil {
		counts["headers"] = doc.Components.Headers.Len()
	}
	if doc.Components.SecuritySchemes != nil {
		counts["securitySchemes"] = doc.Components.SecuritySchemes.Len()
	}
	if doc.Components.Links != nil {
		counts["links"] = doc.Components.Links.Len()
	}
	if doc.Components.Callbacks != nil {
		counts["callbacks"] = doc.Components.Callbacks.Len()
	}
	if doc.Components.PathItems != nil {
		counts["pathItems"] = doc.Components.PathItems.Len()
	}

	return counts
}

// reportCleaningResults reports what was cleaned
func reportCleaningResults(processor *OpenAPIProcessor, before, after map[string]int) {
	totalBefore := 0
	totalAfter := 0
	removedAny := false

	for componentType := range before {
		beforeCount := before[componentType]
		afterCount := after[componentType]
		totalBefore += beforeCount
		totalAfter += afterCount

		if beforeCount > afterCount {
			removed := beforeCount - afterCount
			processor.PrintInfo(fmt.Sprintf("Removed %d unused %s (%d → %d)", removed, componentType, beforeCount, afterCount))
			removedAny = true
		}
	}

	if !removedAny {
		processor.PrintSuccess("No unused components found - document is already clean")
	} else {
		totalRemoved := totalBefore - totalAfter
		processor.PrintSuccess(fmt.Sprintf("Successfully removed %d unused components (%d → %d total)", totalRemoved, totalBefore, totalAfter))
	}
}

// GetCleanCommand returns the clean command for external use
func GetCleanCommand() *cobra.Command {
	return cleanCmd
}
