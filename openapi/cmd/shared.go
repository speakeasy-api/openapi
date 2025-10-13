package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
)

// OpenAPIProcessor handles common OpenAPI document processing operations
type OpenAPIProcessor struct {
	InputFile     string
	OutputFile    string
	WriteToStdout bool
}

// NewOpenAPIProcessor creates a new processor with the given input and output files
func NewOpenAPIProcessor(inputFile, outputFile string, writeInPlace bool) (*OpenAPIProcessor, error) {
	var finalOutputFile string

	if writeInPlace {
		if outputFile != "" {
			return nil, errors.New("cannot specify output file when using --write flag")
		}
		finalOutputFile = inputFile
	} else {
		finalOutputFile = outputFile
	}

	return &OpenAPIProcessor{
		InputFile:     inputFile,
		OutputFile:    finalOutputFile,
		WriteToStdout: finalOutputFile == "",
	}, nil
}

// LoadDocument loads and parses an OpenAPI document from the input file
func (p *OpenAPIProcessor) LoadDocument(ctx context.Context) (*openapi.OpenAPI, []error, error) {
	cleanInputFile := filepath.Clean(p.InputFile)

	// Only print status messages if not writing to stdout (to keep stdout clean for piping)
	if !p.WriteToStdout {
		fmt.Printf("Processing OpenAPI document: %s\n", cleanInputFile)
	}

	// Read the input file
	f, err := os.Open(cleanInputFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open input file: %w", err)
	}
	defer f.Close()

	// Parse the OpenAPI document
	doc, validationErrors, err := openapi.Unmarshal(ctx, f)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal OpenAPI document: %w", err)
	}
	if doc == nil {
		return nil, nil, errors.New("failed to parse OpenAPI document: document is nil")
	}

	return doc, validationErrors, nil
}

// ReportValidationErrors reports validation errors if not writing to stdout
func (p *OpenAPIProcessor) ReportValidationErrors(validationErrors []error) {
	if len(validationErrors) > 0 && !p.WriteToStdout {
		fmt.Printf("⚠️  Found %d validation errors in original document:\n", len(validationErrors))
		for i, validationErr := range validationErrors {
			fmt.Printf("  %d. %s\n", i+1, validationErr.Error())
		}
		fmt.Println()
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

	fmt.Printf("📄 Document written to: %s\n", cleanOutputFile)

	return nil
}

// PrintSuccess prints a success message if not writing to stdout
func (p *OpenAPIProcessor) PrintSuccess(message string) {
	if !p.WriteToStdout {
		fmt.Printf("✅ %s\n", message)
	}
}

// PrintInfo prints an info message if not writing to stdout
func (p *OpenAPIProcessor) PrintInfo(message string) {
	if !p.WriteToStdout {
		fmt.Printf("📋 %s\n", message)
	}
}

// PrintWarning prints a warning message if not writing to stdout
func (p *OpenAPIProcessor) PrintWarning(message string) {
	if !p.WriteToStdout {
		fmt.Printf("⚠️  Warning: %s\n", message)
	}
}
