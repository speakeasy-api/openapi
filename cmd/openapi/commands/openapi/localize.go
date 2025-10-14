package openapi

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/system"
	"github.com/spf13/cobra"
)

var localizeCmd = &cobra.Command{
	Use:   "localize <file> <target-directory>",
	Short: "Localize an OpenAPI specification by copying external references to a target directory",
	Long: `Localize an OpenAPI specification by copying all external reference files to a target directory
and creating a new version of the document with references rewritten to point to the localized files.

The original document is left untouched - all files (including the main document) are copied
to the target directory with updated references, creating a portable document bundle.

Why use Localize?

  - Create portable document bundles: Copy all external dependencies into a single directory
  - Simplify deployment: Package all API definition files together for easy distribution
  - Offline development: Work with API definitions without external file dependencies
  - Version control: Keep all related files in the same repository structure
  - CI/CD pipelines: Ensure all dependencies are available in build environments
  - Documentation generation: Bundle all files needed for complete API documentation

What you'll get:

Before localization:
  main.yaml:
    paths:
      /users:
        get:
          responses:
            '200':
              content:
                application/json:
                  schema:
                    $ref: "./components.yaml#/components/schemas/User"

  components.yaml:
    components:
      schemas:
        User:
          properties:
            address:
              $ref: "./schemas/address.yaml#/Address"

After localization (files copied to target directory):
  target/main.yaml:
    paths:
      /users:
        get:
          responses:
            '200':
              content:
                application/json:
                  schema:
                    $ref: "components.yaml#/components/schemas/User"

  target/components.yaml:
    components:
      schemas:
        User:
          properties:
            address:
              $ref: "schemas-address.yaml#/Address"

  target/schemas-address.yaml:
    Address:
      type: object
      properties:
        street: {type: string}`,
	Args: cobra.ExactArgs(2),
	Run:  runLocalize,
}

var localizeNamingStrategy string

func init() {
	localizeCmd.Flags().StringVar(&localizeNamingStrategy, "naming", "path", "Naming strategy for external files: 'path' (path-based) or 'counter' (counter-based)")
}

func runLocalize(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	inputFile := args[0]
	targetDirectory := args[1]

	if err := localizeOpenAPI(ctx, inputFile, targetDirectory); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func localizeOpenAPI(ctx context.Context, inputFile, targetDirectory string) error {
	cleanInputFile := filepath.Clean(inputFile)
	cleanTargetDir := filepath.Clean(targetDirectory)

	fmt.Printf("Localizing OpenAPI document: %s\n", cleanInputFile)
	fmt.Printf("Target directory: %s\n", cleanTargetDir)

	// Read the input file
	f, err := os.Open(cleanInputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer f.Close()

	// Parse the OpenAPI document
	doc, validationErrors, err := openapi.Unmarshal(ctx, f)
	if err != nil {
		return fmt.Errorf("failed to unmarshal OpenAPI document: %w", err)
	}
	if doc == nil {
		return errors.New("failed to parse OpenAPI document: document is nil")
	}

	// Report validation errors if any
	if len(validationErrors) > 0 {
		fmt.Printf("⚠️  Found %d validation errors in original document:\n", len(validationErrors))
		for i, validationErr := range validationErrors {
			fmt.Printf("  %d. %s\n", i+1, validationErr.Error())
		}
		fmt.Println()
	}

	// Determine naming strategy
	var namingStrategy openapi.LocalizeNamingStrategy
	switch localizeNamingStrategy {
	case "path":
		namingStrategy = openapi.LocalizeNamingPathBased
	case "counter":
		namingStrategy = openapi.LocalizeNamingCounter
	default:
		return fmt.Errorf("invalid naming strategy: %s (must be 'path' or 'counter')", localizeNamingStrategy)
	}

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(cleanTargetDir, 0o750); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Set up localize options
	opts := openapi.LocalizeOptions{
		DocumentLocation: cleanInputFile,
		TargetDirectory:  cleanTargetDir,
		VirtualFS:        &system.FileSystem{},
		NamingStrategy:   namingStrategy,
	}

	// Perform localization (this modifies the doc in memory but doesn't affect the original file)
	if err := openapi.Localize(ctx, doc, opts); err != nil {
		return fmt.Errorf("failed to localize document: %w", err)
	}

	// Write the updated document to the target directory
	outputFile := filepath.Join(cleanTargetDir, filepath.Base(cleanInputFile))
	// Clean the output file path to prevent directory traversal
	cleanOutputFile := filepath.Clean(outputFile)
	outFile, err := os.Create(cleanOutputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	if err := openapi.Marshal(ctx, doc, outFile); err != nil {
		return fmt.Errorf("failed to write localized document: %w", err)
	}

	fmt.Printf("📄 Localized document written to: %s\n", cleanOutputFile)
	fmt.Printf("✅ Localization completed successfully - original document unchanged\n")

	return nil
}

// GetLocalizeCommand returns the localize command for external use
func GetLocalizeCommand() *cobra.Command {
	return localizeCmd
}
