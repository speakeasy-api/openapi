package arazzo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/cmd/openapi/commands/cmdutil"
	"github.com/speakeasy-api/openapi/validation"
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
- Runtime expression validation, including that $steps, $workflows and $inputs
  expressions reference steps, outputs and inputs that exist
- Source description references

By default the OpenAPI, AsyncAPI and Arazzo documents referenced by
sourceDescriptions are loaded (relative URLs are resolved against $self, or the
document's location) and the workflows are checked against them:
- operationId, operationPath, channelPath and workflowId targets exist
- step parameters match the target operation's parameters by name and location
- required parameters and request bodies are supplied
- parameters passed to workflows match their inputs
- AsyncAPI steps only send headers their messages declare

Sources that can't be loaded, including AsyncAPI 2.x sources, are reported as
warnings. Use --skip-sources to only validate the document itself, for example
when working offline. Only errors cause validation to fail.

Stdin is supported — either pipe data directly or use '-' explicitly:
  cat workflow.yaml | openapi arazzo validate
  cat workflow.yaml | openapi arazzo validate -`,
	Args: cmdutil.StdinOrFileArgs(1, 1),
	Run:  runValidate,
}

var validateSkipSources bool

func init() {
	validateCmd.Flags().BoolVar(&validateSkipSources, "skip-sources", false, "don't load source descriptions to validate the document against them")
}

func runValidate(cmd *cobra.Command, args []string) {
	if err := validateArazzo(cmd.Context(), cmdutil.InputFileFromArgs(args), validateSkipSources, os.Stdin, os.Stderr); err != nil {
		cmdutil.Die(err)
	}
}

func validateArazzo(ctx context.Context, file string, skipSources bool, stdin io.Reader, stderr io.Writer) error {
	var data []byte
	location := ""

	if cmdutil.IsStdin(file) {
		fmt.Fprintf(stderr, "Validating Arazzo document from stdin\n")
		var err error
		data, err = io.ReadAll(stdin)
		if err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}
	} else {
		cleanFile := filepath.Clean(file)
		fmt.Fprintf(stderr, "Validating Arazzo document: %s\n", cleanFile)

		var err error
		data, err = os.ReadFile(cleanFile)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		location = cleanFile
	}

	doc, validationErrors, err := arazzo.Unmarshal(ctx, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to unmarshal file: %w", err)
	}

	if !skipSources {
		sources, loadErrs := arazzo.LoadSourceDescriptions(ctx, doc, arazzo.LoadSourcesOptions{DocumentLocation: location})
		validationErrors = append(validationErrors, loadErrs...)
		validationErrors = append(validationErrors, doc.ValidateSources(ctx, sources)...)
		validation.SortValidationErrors(validationErrors)
	}

	errorCount := 0
	for _, validationErr := range validationErrors {
		if severityOf(validationErr).Rank() >= validation.SeverityError.Rank() {
			errorCount++
		}
	}
	otherCount := len(validationErrors) - errorCount

	switch {
	case errorCount > 0:
		fmt.Fprintf(stderr, "❌ Arazzo document is invalid - %d errors", errorCount)
	default:
		fmt.Fprintf(stderr, "✅ Arazzo document is valid - 0 errors")
	}
	if otherCount > 0 {
		fmt.Fprintf(stderr, ", %d warnings or hints", otherCount)
	}

	if len(validationErrors) == 0 {
		fmt.Fprintln(stderr)
		return nil
	}

	fmt.Fprintf(stderr, ":\n\n")
	for i, validationErr := range validationErrors {
		fmt.Fprintf(stderr, "%d. %s\n", i+1, validationErr.Error())
	}

	if errorCount > 0 {
		return errors.New("arazzo document validation failed")
	}

	return nil
}

// severityOf returns the severity of a validation error, errors that aren't validation errors are treated as errors.
func severityOf(err error) validation.Severity {
	var vErr *validation.Error
	if errors.As(err, &vErr) {
		return vErr.Severity
	}
	return validation.SeverityError
}

// GetValidateCommand returns the validate command for external use
func GetValidateCommand() *cobra.Command {
	return validateCmd
}
