package openapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/speakeasy-api/openapi/cmd/openapi/commands/cmdutil"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/spf13/cobra"
)

// IsStdin delegates to cmdutil.IsStdin.
func IsStdin(path string) bool {
	return cmdutil.IsStdin(path)
}

func inputFileFromArgs(args []string) string {
	return cmdutil.InputFileFromArgs(args)
}

func outputFileFromArgs(args []string) string {
	return cmdutil.ArgAt(args, 1, "")
}

func stdinOrFileArgs(minArgs, maxArgs int) cobra.PositionalArgs {
	return cmdutil.StdinOrFileArgs(minArgs, maxArgs)
}

// OpenAPIProcessor handles common OpenAPI document processing operations
type OpenAPIProcessor struct {
	InputFile     string
	OutputFile    string
	ReadFromStdin bool
	WriteToStdout bool

	// Optional overrides for testing — when nil, os.Stdin/os.Stdout/os.Stderr are used.
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func (p *OpenAPIProcessor) stdin() io.Reader {
	if p.Stdin != nil {
		return p.Stdin
	}
	return os.Stdin
}

func (p *OpenAPIProcessor) stdout() io.Writer {
	if p.Stdout != nil {
		return p.Stdout
	}
	return os.Stdout
}

func (p *OpenAPIProcessor) stderr() io.Writer {
	if p.Stderr != nil {
		return p.Stderr
	}
	return os.Stderr
}

// NewOpenAPIProcessor creates a new processor with the given input and output files.
// Pass "-" as inputFile to read from stdin.
func NewOpenAPIProcessor(inputFile, outputFile string, writeInPlace bool) (*OpenAPIProcessor, error) {
	readFromStdin := cmdutil.IsStdin(inputFile)

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
		fmt.Fprintf(p.stderr(), "Processing OpenAPI document from stdin\n")
		reader = io.NopCloser(p.stdin())
	} else {
		cleanInputFile := filepath.Clean(p.InputFile)
		fmt.Fprintf(p.stderr(), "Processing OpenAPI document: %s\n", cleanInputFile)

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
		fmt.Fprintf(p.stderr(), "⚠️  Found %d validation errors in original document:\n", len(validationErrors))
		for i, validationErr := range validationErrors {
			fmt.Fprintf(p.stderr(), "  %d. %s\n", i+1, validationErr.Error())
		}
		fmt.Fprintln(p.stderr())
	}
}

// WriteDocument writes the processed document to the output destination
func (p *OpenAPIProcessor) WriteDocument(ctx context.Context, doc *openapi.OpenAPI) error {
	if p.WriteToStdout {
		// Write to stdout (pipe-friendly)
		return marshaller.Marshal(ctx, doc, p.stdout())
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

	fmt.Fprintf(p.stderr(), "📄 Document written to: %s\n", cleanOutputFile)

	return nil
}

// PrintSuccess prints a success message to stderr.
func (p *OpenAPIProcessor) PrintSuccess(message string) {
	fmt.Fprintf(p.stderr(), "✅ %s\n", message)
}

// PrintInfo prints an info message to stderr.
func (p *OpenAPIProcessor) PrintInfo(message string) {
	fmt.Fprintf(p.stderr(), "📋 %s\n", message)
}

// PrintWarning prints a warning message to stderr.
func (p *OpenAPIProcessor) PrintWarning(message string) {
	fmt.Fprintf(p.stderr(), "⚠️  Warning: %s\n", message)
}
