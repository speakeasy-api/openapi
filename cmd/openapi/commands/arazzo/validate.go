package arazzo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/cmd/openapi/commands/cmdutil"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate <file>",
	Short: "Validate an Arazzo workflow document",
	Long: `Validate an Arazzo workflow document for compliance with the Arazzo Specification.

This command will parse and validate the provided Arazzo document, checking for:
- Structural validity according to the Arazzo Specification
- Required fields and proper data types
- Workflow step dependencies and consistency
- Runtime expression validation
- Source description references

Stdin is supported — either pipe data directly or use '-' explicitly:
  cat workflow.yaml | openapi arazzo validate
  cat workflow.yaml | openapi arazzo validate -`,
	Args: cmdutil.StdinOrFileArgs(1, 1),
	Run:  runValidate,
}

func runValidate(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	file := cmdutil.InputFileFromArgs(args)

	if err := validateArazzo(ctx, file); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func validateArazzo(ctx context.Context, file string) error {
	var reader io.ReadCloser

	if cmdutil.IsStdin(file) {
		fmt.Fprintf(os.Stderr, "Validating Arazzo document from stdin\n")
		reader = io.NopCloser(os.Stdin)
	} else {
		cleanFile := filepath.Clean(file)
		fmt.Fprintf(os.Stderr, "Validating Arazzo document: %s\n", cleanFile)

		f, err := os.Open(cleanFile)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		reader = f
	}
	defer reader.Close()

	_, validationErrors, err := arazzo.Unmarshal(ctx, reader)
	if err != nil {
		return fmt.Errorf("failed to unmarshal file: %w", err)
	}

	if len(validationErrors) == 0 {
		fmt.Fprintf(os.Stderr, "✅ Arazzo document is valid - 0 errors\n")
		return nil
	}

	fmt.Fprintf(os.Stderr, "❌ Arazzo document is invalid - %d errors:\n\n", len(validationErrors))

	for i, validationErr := range validationErrors {
		fmt.Fprintf(os.Stderr, "%d. %s\n", i+1, validationErr.Error())
	}

	return errors.New("arazzo document validation failed")
}

// GetValidateCommand returns the validate command for external use
func GetValidateCommand() *cobra.Command {
	return validateCmd
}
