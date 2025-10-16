package swagger

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	sw "github.com/speakeasy-api/openapi/swagger"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate <file>",
	Short: "Validate a Swagger 2.0 specification document",
	Long: `Validate a Swagger 2.0 (OpenAPI v2) specification document for compliance.

This command will parse and validate the provided Swagger document, checking for:
- Structural validity according to the Swagger 2.0 Specification
- Required fields and proper data types
- Reference resolution and consistency
- Schema validation rules`,
	Args: cobra.ExactArgs(1),
	Run:  runValidate,
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade <input-file> [output-file]",
	Short: "Upgrade a Swagger 2.0 specification to OpenAPI 3.0",
	Long: `Convert a Swagger 2.0 (OpenAPI v2) document to OpenAPI 3.0 (3.0.0).

The upgrade process includes:
- Converting host/basePath/schemes to servers
- Transforming parameters, request bodies, and responses to OAS3 structures
- Mapping definitions to components.schemas
- Migrating securityDefinitions to components.securitySchemes
- Rewriting $ref targets to OAS3 component locations

Output options:
- No output file specified: writes to stdout (pipe-friendly)
- Output file specified: writes to the specified file
- --write flag: writes in-place to the input file`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runUpgrade,
}

var writeInPlace bool

func init() {
	upgradeCmd.Flags().BoolVarP(&writeInPlace, "write", "w", false, "write result in-place to input file")
}

// Apply registers the swagger command group on the provided parent command.
func Apply(rootCmd *cobra.Command) {
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(upgradeCmd)
}

func runValidate(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	file := args[0]

	if err := validateSwagger(ctx, file); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func validateSwagger(ctx context.Context, file string) error {
	cleanFile := filepath.Clean(file)
	fmt.Printf("Validating Swagger document: %s\n", cleanFile)

	f, err := os.Open(cleanFile)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	_, validationErrors, err := sw.Unmarshal(ctx, f)
	if err != nil {
		return fmt.Errorf("failed to unmarshal file: %w", err)
	}

	if len(validationErrors) == 0 {
		fmt.Printf("✅ Swagger document is valid - 0 errors\n")
		return nil
	}

	fmt.Printf("❌ Swagger document is invalid - %d errors:\n\n", len(validationErrors))

	for i, validationErr := range validationErrors {
		fmt.Printf("%d. %s\n", i+1, validationErr.Error())
	}

	return errors.New("swagger document validation failed")
}

// SwaggerProcessor handles IO for converting Swagger -> OpenAPI
type SwaggerProcessor struct {
	InputFile     string
	OutputFile    string
	WriteToStdout bool
}

// NewSwaggerProcessor creates a new processor with the given input and output files
func NewSwaggerProcessor(inputFile, outputFile string, writeInPlace bool) (*SwaggerProcessor, error) {
	var finalOutputFile string

	if writeInPlace {
		if outputFile != "" {
			return nil, errors.New("cannot specify output file when using --write flag")
		}
		finalOutputFile = inputFile
	} else {
		finalOutputFile = outputFile
	}

	return &SwaggerProcessor{
		InputFile:     inputFile,
		OutputFile:    finalOutputFile,
		WriteToStdout: finalOutputFile == "",
	}, nil
}

// LoadDocument loads and parses a Swagger 2.0 document from the input file
func (p *SwaggerProcessor) LoadDocument(ctx context.Context) (*sw.Swagger, []error, error) {
	cleanInputFile := filepath.Clean(p.InputFile)

	// Only print status messages if not writing to stdout (keep stdout clean for piping)
	if !p.WriteToStdout {
		fmt.Printf("Processing Swagger document: %s\n", cleanInputFile)
	}

	f, err := os.Open(cleanInputFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open input file: %w", err)
	}
	defer f.Close()

	doc, validationErrors, err := sw.Unmarshal(ctx, f)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal Swagger document: %w", err)
	}
	if doc == nil {
		return nil, nil, errors.New("failed to parse Swagger document: document is nil")
	}

	return doc, validationErrors, nil
}

// ReportValidationErrors reports validation errors if not writing to stdout
func (p *SwaggerProcessor) ReportValidationErrors(validationErrors []error) {
	if len(validationErrors) > 0 && !p.WriteToStdout {
		fmt.Printf("⚠️  Found %d validation errors in original document:\n", len(validationErrors))
		for i, validationErr := range validationErrors {
			fmt.Printf("  %d. %s\n", i+1, validationErr.Error())
		}
		fmt.Println()
	}
}

// WriteOpenAPIDocument writes the converted OpenAPI document to the output destination
func (p *SwaggerProcessor) WriteOpenAPIDocument(ctx context.Context, doc *openapi.OpenAPI) error {
	if p.WriteToStdout {
		// Write to stdout (pipe-friendly)
		return marshaller.Marshal(ctx, doc, os.Stdout)
	}

	// Write to file
	cleanOutputFile := filepath.Clean(p.OutputFile)
	outFile, err := os.Create(cleanOutputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	if err := marshaller.Marshal(ctx, doc, outFile); err != nil {
		return fmt.Errorf("failed to write document: %w", err)
	}

	fmt.Printf("📄 Document written to: %s\n", cleanOutputFile)

	return nil
}

// PrintSuccess prints a success message if not writing to stdout
func (p *SwaggerProcessor) PrintSuccess(message string) {
	if !p.WriteToStdout {
		fmt.Printf("✅ %s\n", message)
	}
}

// PrintInfo prints an info message if not writing to stdout
func (p *SwaggerProcessor) PrintInfo(message string) {
	if !p.WriteToStdout {
		fmt.Printf("📋 %s\n", message)
	}
}

// PrintWarning prints a warning message if not writing to stdout
func (p *SwaggerProcessor) PrintWarning(message string) {
	if !p.WriteToStdout {
		fmt.Printf("⚠️  Warning: %s\n", message)
	}
}

func runUpgrade(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	inputFile := args[0]

	var outputFile string
	if len(args) > 1 {
		outputFile = args[1]
	}

	processor, err := NewSwaggerProcessor(inputFile, outputFile, writeInPlace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := upgradeSwagger(ctx, processor); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func upgradeSwagger(ctx context.Context, processor *SwaggerProcessor) error {
	// Load the Swagger document
	swDoc, validationErrors, err := processor.LoadDocument(ctx)
	if err != nil {
		return err
	}
	if swDoc == nil {
		return errors.New("failed to parse Swagger document: document is nil")
	}

	// Report validation errors but continue with upgrade
	processor.ReportValidationErrors(validationErrors)

	// Perform the upgrade (Swagger 2.0 -> OpenAPI 3.0)
	oasDoc, err := sw.Upgrade(ctx, swDoc)
	if err != nil {
		return fmt.Errorf("failed to upgrade Swagger document: %w", err)
	}
	if oasDoc == nil {
		return errors.New("upgrade returned a nil document")
	}

	processor.PrintSuccess(fmt.Sprintf("Successfully upgraded to OpenAPI %s", oasDoc.OpenAPI))

	return processor.WriteOpenAPIDocument(ctx, oasDoc)
}
