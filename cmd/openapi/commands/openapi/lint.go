package openapi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/openapi"
	openapiLinter "github.com/speakeasy-api/openapi/openapi/linter"
	"github.com/spf13/cobra"
)

var lintCmd = &cobra.Command{
	Use:   "lint <file>",
	Short: "Lint an OpenAPI specification document",
	Long: `Lint an OpenAPI specification document for style, consistency, and best practices.

This command runs both spec validation and additional lint rules including:
- Path parameter validation
- Operation ID requirements
- Consistent naming conventions
- Security best practices

Output can be formatted as text (default) or JSON.`,
	Args: cobra.ExactArgs(1),
	Run:  runLint,
}

var (
	lintOutputFormat string
	lintRuleset      string
	lintConfigFile   string
	lintDisableRules []string
)

func init() {
	lintCmd.Flags().StringVarP(&lintOutputFormat, "format", "f", "text", "Output format (text, json)")
	lintCmd.Flags().StringVarP(&lintRuleset, "ruleset", "r", "all", "Ruleset to use")
	lintCmd.Flags().StringVarP(&lintConfigFile, "config", "c", "", "Path to lint configuration file")
	lintCmd.Flags().StringSliceVarP(&lintDisableRules, "disable", "d", nil, "Rules to disable")
}

func runLint(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	file := args[0]

	if err := lintOpenAPI(ctx, file); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func lintOpenAPI(ctx context.Context, file string) error {
	cleanFile := filepath.Clean(file)

	// Get absolute path for document location
	absPath, err := filepath.Abs(cleanFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Load the OpenAPI document
	f, err := os.Open(cleanFile)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Unmarshal with validation to get validation errors
	doc, validationErrors, err := openapi.Unmarshal(ctx, f)
	if err != nil {
		return fmt.Errorf("failed to unmarshal file: %w", err)
	}

	// Build linter configuration
	config := buildLintConfig()

	// Create the OpenAPI linter with default rules
	lint := openapiLinter.NewLinter(config)

	// Create document info with location
	docInfo := linter.NewDocumentInfo(doc, absPath)

	// Run linting with validation errors passed in
	output, err := lint.Lint(ctx, docInfo, validationErrors, nil)
	if err != nil {
		return fmt.Errorf("linting failed: %w", err)
	}

	// Format and print output
	switch lintOutputFormat {
	case "json":
		fmt.Println(output.FormatJSON())
	default:
		fmt.Printf("%s\n", cleanFile)
		fmt.Println(output.FormatText())
	}

	// Exit with error code if there are errors
	if output.HasErrors() {
		return fmt.Errorf("linting found %d errors", output.ErrorCount())
	}

	return nil
}

func buildLintConfig() *linter.Config {
	config := linter.NewConfig()

	// Load from config file if specified
	if lintConfigFile != "" {
		loaded, err := linter.LoadConfigFromFile(lintConfigFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		config = loaded
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defaultPath := filepath.Join(homeDir, ".openapi", "lint.yaml")
		loaded, err := linter.LoadConfigFromFile(defaultPath)
		if err == nil {
			config = loaded
		}
	}

	// Disable specified rules
	for _, rule := range lintDisableRules {
		disabled := true
		config.Rules = append(config.Rules, linter.RuleEntry{
			ID:       rule,
			Disabled: &disabled,
		})
	}

	// Set output format
	switch lintOutputFormat {
	case "json":
		config.OutputFormat = linter.OutputFormatJSON
	default:
		config.OutputFormat = linter.OutputFormatText
	}

	return config
}

func ptr[T any](v T) *T {
	return &v
}
