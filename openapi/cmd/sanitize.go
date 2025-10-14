package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/spf13/cobra"
)

var sanitizeCmd = &cobra.Command{
	Use:   "sanitize <input-file> [output-file]",
	Short: "Remove unwanted elements from an OpenAPI specification",
	Long: `Sanitize an OpenAPI specification by removing unwanted elements such as vendor extensions,
unused components, and unknown properties.

This command provides comprehensive cleanup of OpenAPI documents to prepare them for
distribution, standardization, or sharing. By default, it performs aggressive cleanup
by removing all extensions and unused components.

Default behavior (no config):
- Removes ALL x-* extensions throughout the document
- Removes unused components (schemas, responses, parameters, etc.)
- Removes unknown properties not in the OpenAPI specification

With a configuration file, you can:
- Selectively remove extensions by pattern (e.g., x-go-*, x-internal-*)
- Keep unused components if needed
- Keep unknown properties if needed

What gets sanitized by default:
- All x-* vendor extensions (info, paths, operations, schemas, etc.)
- Unused schemas in components/schemas
- Unused responses in components/responses
- Unused parameters in components/parameters
- Unused examples in components/examples
- Unused request bodies in components/requestBodies
- Unused headers in components/headers
- Unused security schemes in components/securitySchemes
- Unused links in components/links
- Unused callbacks in components/callbacks
- Unused path items in components/pathItems
- Unknown properties not defined in OpenAPI spec

Benefits of sanitization:
- **Standards compliance**: Remove vendor-specific extensions for clean, standard specs
- **Clean distribution**: Prepare specifications for public sharing or publishing
- **Reduce document size**: Remove unnecessary extensions and unused components
- **Selective cleanup**: Use patterns to target specific extension families
- **Flexible control**: Config file allows fine-grained control over what to keep

Configuration file format (YAML):

  # Remove only specific extension patterns (if not set, removes ALL extensions)
  extensionPatterns:
    - "x-go-*"
    - "x-internal-*"
  
  # Keep unused components (default: false, removes them)
  keepUnusedComponents: true
  
  # Keep unknown properties (default: false, removes them)
  keepUnknownProperties: true

Output options:
- No output file specified: writes to stdout (pipe-friendly)
- Output file specified: writes to the specified file
- --write flag: writes in-place to the input file

Examples:
  # Default sanitization (remove all extensions and unused components)
  openapi spec sanitize ./api.yaml

  # Sanitize and write to new file
  openapi spec sanitize ./api.yaml ./clean-api.yaml

  # Sanitize in-place
  openapi spec sanitize -w ./api.yaml

  # Use config file for selective sanitization
  openapi spec sanitize --config sanitize-config.yaml ./api.yaml

  # Combine config and output options
  openapi spec sanitize --config sanitize-config.yaml -w ./api.yaml`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runSanitize,
}

var (
	sanitizeWriteInPlace bool
	sanitizeConfigFile   string
)

func init() {
	sanitizeCmd.Flags().BoolVarP(&sanitizeWriteInPlace, "write", "w", false, "write result in-place to input file")
	sanitizeCmd.Flags().StringVarP(&sanitizeConfigFile, "config", "c", "", "path to sanitize configuration file")
}

func runSanitize(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	inputFile := args[0]

	var outputFile string
	if len(args) > 1 {
		outputFile = args[1]
	}

	processor, err := NewOpenAPIProcessor(inputFile, outputFile, sanitizeWriteInPlace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := sanitizeOpenAPI(ctx, processor); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func sanitizeOpenAPI(ctx context.Context, processor *OpenAPIProcessor) error {
	// Load the OpenAPI document
	doc, validationErrors, err := processor.LoadDocument(ctx)
	if err != nil {
		return err
	}
	if doc == nil {
		return errors.New("failed to parse OpenAPI document: document is nil")
	}

	// Report validation errors but continue with sanitization
	processor.ReportValidationErrors(validationErrors)

	// Load sanitize options from config file if provided
	var opts *openapi.SanitizeOptions
	if sanitizeConfigFile != "" {
		opts, err = openapi.LoadSanitizeConfigFromFile(sanitizeConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load config file: %w", err)
		}
		processor.PrintInfo("Using configuration from " + sanitizeConfigFile)
	}

	// Perform the sanitization
	result, err := openapi.Sanitize(ctx, doc, opts)
	if err != nil {
		return fmt.Errorf("failed to sanitize OpenAPI document: %w", err)
	}

	// Report any warnings
	for _, warning := range result.Warnings {
		processor.PrintWarning(warning)
	}

	// Report success
	reportSanitizationResults(processor, opts)

	return processor.WriteDocument(ctx, doc)
}

// reportSanitizationResults reports the sanitization operation
func reportSanitizationResults(processor *OpenAPIProcessor, opts *openapi.SanitizeOptions) {
	var messages []string

	// Determine what was done with extensions
	if opts == nil || opts.ExtensionPatterns == nil {
		// nil patterns = remove all extensions (default)
		messages = append(messages, "removed all extensions")
	} else if len(opts.ExtensionPatterns) == 0 {
		// empty slice = keep all extensions (explicit)
		messages = append(messages, "kept all extensions")
	} else {
		// specific patterns = remove matching extensions
		messages = append(messages, fmt.Sprintf("removed extensions matching %v", opts.ExtensionPatterns))
	}

	// Determine what was done with components
	if opts == nil || !opts.KeepUnusedComponents {
		messages = append(messages, "removed unused components")
	} else {
		messages = append(messages, "kept all components")
	}

	// Determine what was done with unknown properties
	if opts != nil && opts.KeepUnknownProperties {
		messages = append(messages, "kept unknown properties")
	}

	// Build the success message
	successMsg := "Successfully sanitized document ("
	for i, msg := range messages {
		if i > 0 {
			successMsg += ", "
		}
		successMsg += msg
	}
	successMsg += ")"

	processor.PrintSuccess(successMsg)
}

// GetSanitizeCommand returns the sanitize command for external use
func GetSanitizeCommand() *cobra.Command {
	return sanitizeCmd
}
