package openapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/spf13/cobra"
)

// StdinIndicator is the conventional Unix indicator to read from stdin.
const StdinIndicator = "-"

// IsStdin returns true if the given path indicates stdin should be used.
func IsStdin(path string) bool {
	return path == StdinIndicator
}

// StdinIsPiped returns true when stdin is connected to a pipe (not a terminal),
// meaning data is being piped in from another command or a file redirect.
func StdinIsPiped() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) == 0
}

// inputFileFromArgs returns the input file from args, or "-" if stdin should
// be used. It extracts the first positional arg or detects piped stdin.
func inputFileFromArgs(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return StdinIndicator
}

// stdinOrFileArgs returns a cobra arg validator that accepts minArgs..maxArgs
// when a file is given, but also allows zero args when stdin is piped.
func stdinOrFileArgs(minArgs, maxArgs int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			if StdinIsPiped() {
				return nil
			}
			return fmt.Errorf("requires at least %d arg(s), or pipe data to stdin", minArgs)
		}
		if len(args) < minArgs {
			return fmt.Errorf("requires at least %d arg(s), only received %d", minArgs, len(args))
		}
		if maxArgs >= 0 && len(args) > maxArgs {
			return fmt.Errorf("accepts at most %d arg(s), received %d", maxArgs, len(args))
		}
		return nil
	}
}

// OpenAPIProcessor handles common OpenAPI document processing operations
type OpenAPIProcessor struct {
	InputFile     string
	OutputFile    string
	ReadFromStdin bool
	WriteToStdout bool
}

// NewOpenAPIProcessor creates a new processor with the given input and output files.
// Pass "-" as inputFile to read from stdin.
func NewOpenAPIProcessor(inputFile, outputFile string, writeInPlace bool) (*OpenAPIProcessor, error) {
	readFromStdin := IsStdin(inputFile)

	if writeInPlace {
		if readFromStdin {
			return nil, errors.New("cannot use --write flag when reading from stdin")
		}
		if outputFile != "" {
			return nil, errors.New("cannot specify output file when using --write flag")
		}
		outputFile = inputFile
	}

	return &OpenAPIProcessor{
		InputFile:     inputFile,
		OutputFile:    outputFile,
		ReadFromStdin: readFromStdin,
		WriteToStdout: outputFile == "",
	}, nil
}

// LoadDocument loads and parses an OpenAPI document from the input file or stdin.
func (p *OpenAPIProcessor) LoadDocument(ctx context.Context) (*openapi.OpenAPI, []error, error) {
	var reader io.ReadCloser

	if p.ReadFromStdin {
		fmt.Fprintf(os.Stderr, "Processing OpenAPI document from stdin\n")
		reader = io.NopCloser(os.Stdin)
	} else {
		cleanInputFile := filepath.Clean(p.InputFile)
		fmt.Fprintf(os.Stderr, "Processing OpenAPI document: %s\n", cleanInputFile)

		f, err := os.Open(cleanInputFile)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open input file: %w", err)
		}
		reader = f
	}
	defer reader.Close()

	// Parse the OpenAPI document
	doc, validationErrors, err := openapi.Unmarshal(ctx, reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal OpenAPI document: %w", err)
	}
	if doc == nil {
		return nil, nil, errors.New("failed to parse OpenAPI document: document is nil")
	}

	return doc, validationErrors, nil
}

// ReportValidationErrors reports validation errors to stderr.
func (p *OpenAPIProcessor) ReportValidationErrors(validationErrors []error) {
	if len(validationErrors) > 0 {
		fmt.Fprintf(os.Stderr, "⚠️  Found %d validation errors in original document:\n", len(validationErrors))
		for i, validationErr := range validationErrors {
			fmt.Fprintf(os.Stderr, "  %d. %s\n", i+1, validationErr.Error())
		}
		fmt.Fprintln(os.Stderr)
	}
}

// WriteDocument writes the processed document to the output destination
func (p *OpenAPIProcessor) WriteDocument(ctx context.Context, doc *openapi.OpenAPI) error {
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

	fmt.Fprintf(os.Stderr, "📄 Document written to: %s\n", cleanOutputFile)

	return nil
}

// PrintSuccess prints a success message to stderr.
func (p *OpenAPIProcessor) PrintSuccess(message string) {
	fmt.Fprintf(os.Stderr, "✅ %s\n", message)
}

// PrintInfo prints an info message to stderr.
func (p *OpenAPIProcessor) PrintInfo(message string) {
	fmt.Fprintf(os.Stderr, "📋 %s\n", message)
}

// PrintWarning prints a warning message to stderr.
func (p *OpenAPIProcessor) PrintWarning(message string) {
	fmt.Fprintf(os.Stderr, "⚠️  Warning: %s\n", message)
}
