package openapi

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/spf13/cobra"
)

const (
	reset = "\x1b[0m"

	// 256-color (widely supported). Tweak these if your terminal theme needs it.
	bgBlock  = "\x1b[48;5;236m" // dark gray background
	fgBlock  = "\x1b[38;5;252m" // light foreground for contrast
	fgBorder = "\x1b[38;5;240m" // dimmer border color
)

var optimizeCmd = &cobra.Command{
	Use:   "optimize <input-file> [output-file]",
	Short: "Optimize an OpenAPI specification by deduplicating inline schemas",
	Long: `Optimize an OpenAPI specification by finding duplicate inline schemas and extracting them to reusable components.

Stdin is supported — either pipe data directly or use '-' explicitly:
  cat spec.yaml | openapi spec optimize --non-interactive
  cat spec.yaml | openapi spec optimize - --non-interactive

This command analyzes an OpenAPI document to identify inline JSON schemas that appear multiple times
with identical content and replaces them with references to newly created or existing components.

What gets optimized:
- Duplicate object schemas with identical properties
- Duplicate enum schemas with identical values
- Duplicate oneOf/allOf/anyOf schemas with identical structure
- Duplicate conditional schemas (if/then/else)
- Duplicate schemas with complex patterns (additionalProperties, patternProperties, etc.)

What is preserved:
- Existing component schemas (not modified or replaced)
- Simple type schemas (string, number, boolean) - not extracted
- Schemas that appear only once (no duplication)
- Top-level component schemas remain unchanged

Interactive mode (default):
- Shows each duplicate schema and prompts for a custom name
- Displays the schema content in a formatted code block
- Allows you to provide meaningful names instead of generated ones
- Press Enter to accept the suggested name

Non-interactive mode (--non-interactive):
- Uses automatically generated names based on schema content hash
- No user prompts - suitable for automation and CI/CD

Benefits of optimization:
- Reduce document size by eliminating duplicate schema definitions
- Improve maintainability by centralizing schema definitions
- Enhance reusability by making schemas available as components
- Optimize tooling performance with smaller, cleaner documents
- Follow OpenAPI best practices for schema organization

Output options:
- No output file specified: writes to stdout (pipe-friendly)
- Output file specified: writes to the specified file
- --write flag: writes in-place to the input file

Examples:
  # Interactive optimization (default)
  openapi spec optimize api.yaml

  # Non-interactive optimization
  openapi spec optimize api.yaml --non-interactive

  # Optimize and write to a new file
  openapi spec optimize api.yaml optimized-api.yaml

  # Optimize in-place
  openapi spec optimize api.yaml --write`,
	Args: stdinOrFileArgs(1, 2),
	Run:  runOptimize,
}

var (
	optimizeWriteInPlace   bool
	optimizeNonInteractive bool
)

func init() {
	optimizeCmd.Flags().BoolVarP(&optimizeWriteInPlace, "write", "w", false, "write result in-place to input file")
	optimizeCmd.Flags().BoolVar(&optimizeNonInteractive, "non-interactive", false, "run in non-interactive mode (no prompts)")
}

func runOptimize(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	inputFile := inputFileFromArgs(args)

	// Interactive mode reads user input from stdin, which conflicts with
	// document data already being piped via stdin.
	if IsStdin(inputFile) && !optimizeNonInteractive {
		fmt.Fprintf(os.Stderr, "Error: interactive mode is not supported when reading from stdin; use --non-interactive\n")
		os.Exit(1)
	}

	outputFile := outputFileFromArgs(args)

	processor, err := NewOpenAPIProcessor(inputFile, outputFile, optimizeWriteInPlace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := optimizeOpenAPI(ctx, processor); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func optimizeOpenAPI(ctx context.Context, processor *OpenAPIProcessor) error {
	// Load the OpenAPI document
	doc, validationErrors, err := processor.LoadDocument(ctx)
	if err != nil {
		return err
	}
	if doc == nil {
		return errors.New("failed to parse OpenAPI document: document is nil")
	}

	// Report validation errors but continue with optimization
	processor.ReportValidationErrors(validationErrors)

	// Count components before optimization
	componentCounts := countComponents(doc)

	// Create the appropriate callback based on mode
	var nameCallback openapi.OptimizeNameCallback
	if !optimizeNonInteractive {
		nameCallback = createInteractiveCallback(ctx, processor)
	}

	// Perform the optimization
	if err := openapi.Optimize(ctx, doc, nameCallback); err != nil {
		return fmt.Errorf("failed to optimize OpenAPI document: %w", err)
	}

	// Count components after optimization and report
	newComponentCounts := countComponents(doc)
	reportOptimizationResults(processor, componentCounts, newComponentCounts)

	return processor.WriteDocument(ctx, doc)
}

// boxedCode renders code in a beautiful box with ANSI colors
func boxedCode(code string) string {
	// normalize line endings and expand tabs (optional)
	lines := strings.Split(strings.ReplaceAll(code, "\r\n", "\n"), "\n")
	for i := range lines {
		lines[i] = strings.ReplaceAll(lines[i], "\t", "  ")
	}

	// measure visible width (runes, good enough for ASCII/YAML)
	maxWidth := 0
	for _, l := range lines {
		if w := utf8.RuneCountInString(l); w > maxWidth {
			maxWidth = w
		}
	}

	// padding inside the box
	padLeft, padRight := 2, 2
	inner := maxWidth + padLeft + padRight

	// build top border
	top := fmt.Sprintf("%s┌%s┐%s",
		fgBorder,
		strings.Repeat("─", inner),
		reset,
	)

	// build bottom border
	bot := fmt.Sprintf("%s└%s┘%s",
		fgBorder,
		strings.Repeat("─", inner),
		reset,
	)

	// build body lines with background
	var b strings.Builder
	b.WriteString(top + "\n")

	// optional empty padding row
	empty := fmt.Sprintf("%s│%s%s%s%s│%s",
		fgBorder, bgBlock+fgBlock, strings.Repeat(" ", inner), reset, fgBorder, reset,
	)
	b.WriteString(empty + "\n")

	for _, l := range lines {
		// right pad each line to maxWidth width
		spaces := strings.Repeat(" ", maxWidth-utf8.RuneCountInString(l))
		b.WriteString(fmt.Sprintf(
			"%s│%s%s%s%s%s%s%s│%s\n",
			fgBorder, bgBlock+fgBlock, strings.Repeat(" ", padLeft),
			l, spaces, strings.Repeat(" ", padRight), reset, fgBorder, reset,
		))
	}

	b.WriteString(empty + "\n")
	b.WriteString(bot)

	return b.String()
}

// createInteractiveCallback creates a callback that prompts the user for schema names
func createInteractiveCallback(ctx context.Context, processor *OpenAPIProcessor) openapi.OptimizeNameCallback {
	reader := bufio.NewReader(os.Stdin)

	return func(suggestedName, hash string, locations []string, schema *oas3.JSONSchema[oas3.Referenceable]) string {
		// Display schema information to stderr (interactive prompts)
		fmt.Fprint(os.Stderr, "\n"+strings.Repeat("=", 80)+"\n")
		fmt.Fprintf(os.Stderr, "Found duplicate schema at %d locations:\n", len(locations))
		for i, location := range locations {
			fmt.Fprintf(os.Stderr, "  %d. %s\n", i+1, location)
		}
		fmt.Fprintf(os.Stderr, "\nSchema content:\n")
		fmt.Fprint(os.Stderr, strings.Repeat("-", 40)+"\n")

		// Convert schema to YAML for display using marshaller
		if schema != nil && !schema.IsReference() && schema.GetLeft() != nil {
			var schemaBuilder strings.Builder
			err := marshaller.Marshal(ctx, schema.GetLeft(), &schemaBuilder)
			if err == nil {
				// Display the schema in a beautiful code block
				fmt.Fprintln(os.Stderr, boxedCode(schemaBuilder.String()))
			} else {
				fmt.Fprintf(os.Stderr, "  (Unable to display schema: %v)\n", err)
			}
		}

		fmt.Fprint(os.Stderr, strings.Repeat("-", 40)+"\n")
		fmt.Fprintf(os.Stderr, "Suggested name: %s\n", suggestedName)
		fmt.Fprintf(os.Stderr, "Enter custom name (or press Enter to use suggested): ")

		// Read user input
		input, err := reader.ReadString('\n')
		if err != nil {
			processor.PrintInfo(fmt.Sprintf("⚠️  Error reading input: %v, using suggested name", err))
			return suggestedName
		}

		// Clean up the input
		customName := strings.TrimSpace(input)
		if customName == "" {
			processor.PrintInfo("Using suggested name: " + suggestedName)
			return suggestedName
		}

		// Validate the custom name (basic validation)
		if !isValidComponentName(customName) {
			processor.PrintInfo(fmt.Sprintf("⚠️  Invalid component name '%s', using suggested name: %s", customName, suggestedName))
			return suggestedName
		}

		processor.PrintSuccess("Using custom name: " + customName)
		return customName
	}
}

// isValidComponentName performs basic validation on component names
func isValidComponentName(name string) bool {
	if name == "" {
		return false
	}

	// Component names should be valid identifiers
	// Allow letters, numbers, underscores, and hyphens
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') &&
			(r < '0' || r > '9') && r != '_' && r != '-' {
			return false
		}
	}

	return true
}

// reportOptimizationResults reports what was optimized
func reportOptimizationResults(processor *OpenAPIProcessor, before, after map[string]int) {
	beforeSchemas := before["schemas"]
	afterSchemas := after["schemas"]

	if afterSchemas > beforeSchemas {
		added := afterSchemas - beforeSchemas
		processor.PrintSuccess(fmt.Sprintf("Successfully optimized document: added %d new schema components (%d → %d schemas)", added, beforeSchemas, afterSchemas))
		processor.PrintInfo("Duplicate inline schemas have been extracted to reusable components")
	} else {
		processor.PrintSuccess("Document analyzed - no duplicate schemas found to optimize")
	}
}

// GetOptimizeCommand returns the optimize command for external use
func GetOptimizeCommand() *cobra.Command {
	return optimizeCmd
}
