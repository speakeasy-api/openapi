package openapi

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/speakeasy-api/openapi/openapi"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate <file>",
	Short: "Validate an OpenAPI specification document",
	Long: `Validate an OpenAPI specification document for compliance with the OpenAPI Specification.

This command will parse and validate the provided OpenAPI document, checking for:
- Structural validity according to the OpenAPI Specification
- Required fields and proper data types
- Reference resolution and consistency
- Schema validation rules`,
	Args: cobra.ExactArgs(1),
	Run:  runValidate,
}

func runValidate(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	file := args[0]
	start := time.Now()

	err := validateOpenAPI(ctx, file)
	reportElapsed(os.Stderr, "Validation", time.Since(start))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func validateOpenAPI(ctx context.Context, file string) error {
	cleanFile := filepath.Clean(file)
	fmt.Printf("Validating OpenAPI document: %s\n", cleanFile)

	f, err := os.Open(cleanFile)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	_, validationErrors, err := openapi.Unmarshal(ctx, f)
	if err != nil {
		return fmt.Errorf("failed to unmarshal file: %w", err)
	}

	if len(validationErrors) == 0 {
		fmt.Printf("✅ OpenAPI document is valid - 0 errors\n")
		return nil
	}

	fmt.Printf("❌ OpenAPI document is invalid - %d errors:\n\n", len(validationErrors))
	fmt.Print(formatValidationErrors(validationErrors))

	return errors.New("openAPI document validation failed")
}

func formatValidationErrors(validationErrors []error) string {
	var sb strings.Builder
	indexWidth := len(strconv.Itoa(len(validationErrors)))

	for i, validationErr := range validationErrors {
		fmt.Fprintf(&sb, "%*d. %s\n", indexWidth, i+1, validationErr.Error())
	}

	return sb.String()
}

// GetValidateCommand returns the validate command for external use
func GetValidateCommand() *cobra.Command {
	return validateCmd
}
