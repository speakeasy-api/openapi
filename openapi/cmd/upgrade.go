package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade <input-file> [output-file]",
	Short: "Upgrade an OpenAPI specification to the latest supported version",
	Long: `Upgrade an OpenAPI specification document to the latest supported version (3.1.1).

This command will upgrade OpenAPI documents from:
- OpenAPI 3.0.x versions to 3.1.1 (always)
- OpenAPI 3.1.x versions to 3.1.1 (by default)
- Use --minor-only to only upgrade minor versions (3.0.x to 3.1.1, but skip 3.1.x versions)

The upgrade process includes:
- Updating the OpenAPI version field
- Converting nullable properties to proper JSON Schema format
- Updating schema validation rules
- Maintaining backward compatibility where possible

Output options:
- No output file specified: writes to stdout (pipe-friendly)
- Output file specified: writes to the specified file
- --write flag: writes in-place to the input file`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runUpgrade,
}

var (
	minorOnly    bool
	writeInPlace bool
)

func init() {
	upgradeCmd.Flags().BoolVar(&minorOnly, "minor-only", false, "only upgrade minor versions (3.0.x to 3.1.1, skip 3.1.x versions)")
	upgradeCmd.Flags().BoolVarP(&writeInPlace, "write", "w", false, "write result in-place to input file")
}

func runUpgrade(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	inputFile := args[0]

	var outputFile string
	if writeInPlace {
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "Error: cannot specify output file when using --write flag\n")
			os.Exit(1)
		}
		outputFile = inputFile
	} else if len(args) > 1 {
		outputFile = args[1]
	}

	if err := upgradeOpenAPI(ctx, inputFile, outputFile, minorOnly); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func upgradeOpenAPI(ctx context.Context, inputFile, outputFile string, minorOnly bool) error {
	cleanInputFile := filepath.Clean(inputFile)

	// Only print status messages if not writing to stdout (to keep stdout clean for piping)
	writeToStdout := outputFile == ""
	if !writeToStdout {
		fmt.Printf("Upgrading OpenAPI document: %s\n", cleanInputFile)
	}

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

	// Report validation errors but continue with upgrade (only if not writing to stdout)
	if len(validationErrors) > 0 && !writeToStdout {
		fmt.Printf("⚠️  Found %d validation errors in original document:\n", len(validationErrors))
		for i, validationErr := range validationErrors {
			fmt.Printf("  %d. %s\n", i+1, validationErr.Error())
		}
		fmt.Println()
	}

	// Prepare upgrade options
	var opts []openapi.Option[openapi.UpgradeOptions]
	if !minorOnly {
		// By default, upgrade all versions including patch versions (3.1.x to 3.1.1)
		opts = append(opts, openapi.WithUpgradeSamePatchVersion())
	}
	// When minorOnly is true, only 3.0.x versions will be upgraded to 3.1.1
	// 3.1.x versions will be skipped unless they need minor version upgrade

	// Perform the upgrade
	originalVersion := doc.OpenAPI
	upgraded, err := openapi.Upgrade(ctx, doc, opts...)
	if err != nil {
		return fmt.Errorf("failed to upgrade OpenAPI document: %w", err)
	}

	if !upgraded {
		if !writeToStdout {
			fmt.Printf("📋 No upgrade needed - document is already at version %s\n", originalVersion)
		}
		// Still output the document even if no upgrade was needed
		return writeOutput(ctx, doc, outputFile, writeToStdout)
	}

	if !writeToStdout {
		fmt.Printf("✅ Successfully upgraded from %s to %s\n", originalVersion, doc.OpenAPI)
	}

	return writeOutput(ctx, doc, outputFile, writeToStdout)
}

func writeOutput(ctx context.Context, doc *openapi.OpenAPI, outputFile string, writeToStdout bool) error {
	if writeToStdout {
		// Write to stdout (pipe-friendly)
		return marshaller.Marshal(ctx, doc, os.Stdout)
	}

	// Write to file
	cleanOutputFile := filepath.Clean(outputFile)
	outFile, err := os.Create(cleanOutputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	if err := marshaller.Marshal(ctx, doc, outFile); err != nil {
		return fmt.Errorf("failed to write upgraded document: %w", err)
	}

	if cleanOutputFile == filepath.Clean(outputFile) {
		fmt.Printf("📄 Upgraded document written to: %s\n", cleanOutputFile)
	}

	return nil
}

// GetUpgradeCommand returns the upgrade command for external use
func GetUpgradeCommand() *cobra.Command {
	return upgradeCmd
}
