package openapi

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"sync"

	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/linter/fix"
	"github.com/speakeasy-api/openapi/openapi"
	openapiLinter "github.com/speakeasy-api/openapi/openapi/linter"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/spf13/cobra"

	// Enable custom rules support
	_ "github.com/speakeasy-api/openapi/openapi/linter/customrules"
)

var lintCmd = &cobra.Command{
	Use:   "lint <file>",
	Short: "Lint an OpenAPI specification document",
	Long: `Lint an OpenAPI specification document for style, consistency, and best practices.

This command runs both spec validation and additional lint rules including:
- Path parameter validation
- Operation ID requirements
- Consistent naming conventions
- Security best practices (OWASP)

Use '-' as the file argument to read from stdin:
  cat spec.yaml | openapi spec lint -

Note: --fix and --fix-interactive are not supported when reading from stdin.

CONFIGURATION:

By default, the linter looks for a configuration file at ~/.openapi/lint.yaml.
Use --config to specify a custom configuration file.

Available rulesets: all (default), recommended, security

Example configuration (lint.yaml):

  extends: recommended

  rules:
    - id: operation-operationId
      severity: error
    - id: some-rule
      disabled: true

  custom_rules:
    paths:
      - ./rules/*.ts

CUSTOM RULES:

Write custom linting rules in TypeScript or JavaScript. Install the types package
in your rules directory:

  npm install @speakeasy-api/openapi-linter-types

Then configure the paths in your lint.yaml under custom_rules.paths.

AUTOFIXING:

Use --fix to automatically apply non-interactive fixes. Use --fix-interactive to
also be prompted for fixes that require user input (choosing values, entering text).
Use --dry-run with either flag to preview what would be changed without modifying the file.

See the full documentation at:
https://github.com/speakeasy-api/openapi/blob/main/cmd/openapi/commands/openapi/README.md#lint`,
	Args:    cobra.ExactArgs(1),
	PreRunE: validateLintFlags,
	Run:     runLint,
}

var (
	lintOutputFormat   string
	lintRuleset        string
	lintConfigFile     string
	lintDisableRules   []string
	lintSummary        bool
	lintFix            bool
	lintFixInteractive bool
	lintDryRun         bool
)

func init() {
	lintCmd.Flags().StringVarP(&lintOutputFormat, "format", "f", "text", "Output format: text or json")
	lintCmd.Flags().StringVarP(&lintRuleset, "ruleset", "r", "all", "Ruleset to use (default loads from config)")
	lintCmd.Flags().StringVarP(&lintConfigFile, "config", "c", "", "Path to lint config file (default: ~/.openapi/lint.yaml)")
	lintCmd.Flags().StringSliceVarP(&lintDisableRules, "disable", "d", nil, "Rule IDs to disable (can be repeated)")
	lintCmd.Flags().BoolVar(&lintSummary, "summary", false, "Print a per-rule summary table of findings")
	lintCmd.Flags().BoolVar(&lintFix, "fix", false, "Automatically apply non-interactive fixes and write back")
	lintCmd.Flags().BoolVar(&lintFixInteractive, "fix-interactive", false, "Apply all fixes, prompting for interactive ones")
	lintCmd.Flags().BoolVar(&lintDryRun, "dry-run", false, "Show what fixes would be applied without changing the file (requires --fix or --fix-interactive)")
}

func validateLintFlags(_ *cobra.Command, args []string) error {
	if lintFix && lintFixInteractive {
		return fmt.Errorf("--fix and --fix-interactive are mutually exclusive")
	}
	if lintDryRun && !lintFix && !lintFixInteractive {
		return fmt.Errorf("--dry-run requires --fix or --fix-interactive")
	}
	if len(args) > 0 && IsStdin(args[0]) && (lintFix || lintFixInteractive) {
		return fmt.Errorf("--fix and --fix-interactive are not supported when reading from stdin")
	}
	return nil
}

func runLint(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	file := args[0]
	start := time.Now()

	err := lintOpenAPI(ctx, file)
	reportElapsed(os.Stderr, "Linting", time.Since(start))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func lintOpenAPI(ctx context.Context, file string) error {
	fromStdin := IsStdin(file)

	var reader io.ReadCloser
	var absPath string

	if fromStdin {
		fmt.Fprintf(os.Stderr, "Linting OpenAPI document from stdin\n")
		reader = io.NopCloser(os.Stdin)
		absPath = "stdin"
	} else {
		cleanFile := filepath.Clean(file)

		// Get absolute path for document location
		var err error
		absPath, err = filepath.Abs(cleanFile)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Linting OpenAPI document: %s\n", cleanFile)

		f, err := os.Open(cleanFile)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		reader = f
	}
	defer reader.Close()

	// Unmarshal with validation to get validation errors
	doc, validationErrors, err := openapi.Unmarshal(ctx, reader)
	if err != nil {
		return fmt.Errorf("failed to unmarshal file: %w", err)
	}

	// Build linter configuration
	config := buildLintConfig()

	// Create the OpenAPI linter with default rules
	lint, err := openapiLinter.NewLinter(config)
	if err != nil {
		return fmt.Errorf("failed to create linter: %w", err)
	}

	// Create document info with location
	docInfo := linter.NewDocumentInfo(doc, absPath)

	// Run linting with validation errors passed in
	output, err := lint.Lint(ctx, docInfo, validationErrors, nil)
	if err != nil {
		return fmt.Errorf("linting failed: %w", err)
	}

	// Determine fix mode (already validated that fix is not used with stdin)
	fixOpts := fix.Options{Mode: fix.ModeNone, DryRun: lintDryRun}
	switch {
	case lintFixInteractive:
		fixOpts.Mode = fix.ModeInteractive
	case lintFix:
		fixOpts.Mode = fix.ModeAuto
	}

	if fixOpts.Mode != fix.ModeNone {
		cleanFile := filepath.Clean(file)
		if err := applyFixes(ctx, fixOpts, doc, output, cleanFile); err != nil {
			return err
		}

		// Re-lint after applying fixes (unless dry-run) to get accurate remaining count
		if !lintDryRun {
			// Reload and re-lint the fixed document
			reloadedF, err := os.Open(cleanFile)
			if err != nil {
				return fmt.Errorf("failed to reopen file after fix: %w", err)
			}
			defer reloadedF.Close()

			reloadedDoc, reloadedValErrs, err := openapi.Unmarshal(ctx, reloadedF)
			if err != nil {
				return fmt.Errorf("failed to unmarshal fixed file: %w", err)
			}

			reloadedDocInfo := linter.NewDocumentInfo(reloadedDoc, absPath)
			output, err = lint.Lint(ctx, reloadedDocInfo, reloadedValErrs, nil)
			if err != nil {
				return fmt.Errorf("re-linting failed: %w", err)
			}
		}
	}

	// Format and print output to stdout (this is the data output)
	displayFile := file
	if fromStdin {
		displayFile = "stdin"
	}
	switch lintOutputFormat {
	case "json":
		fmt.Println(output.FormatJSON())
	default:
		fmt.Printf("%s\n", displayFile)
		fmt.Println(output.FormatText())
	}

	// Print per-rule summary if requested
	if lintSummary {
		fmt.Println(output.FormatSummary())
	}

	// Exit with error code if there are errors
	if output.HasErrors() {
		return fmt.Errorf("linting found %d errors", output.ErrorCount())
	}

	return nil
}

func applyFixes(ctx context.Context, fixOpts fix.Options, doc *openapi.OpenAPI, output *linter.Output, cleanFile string) error {
	// Create prompter lazily for interactive mode — only initialized when
	// an interactive fix is actually encountered, avoiding unnecessary setup
	// when all fixes are non-interactive.
	var prompter validation.Prompter
	if fixOpts.Mode == fix.ModeInteractive {
		prompter = &lazyPrompter{}
	}

	engine := fix.NewEngine(fixOpts, prompter, fix.NewFixRegistry())
	result, err := engine.ProcessErrors(ctx, doc, output.Results)
	if err != nil {
		return fmt.Errorf("fix processing failed: %w", err)
	}

	// Report fix results to stderr
	reportFixResults(result, fixOpts.DryRun)

	// Write modified document back if any fixes were applied (and not dry-run)
	if len(result.Applied) > 0 && !fixOpts.DryRun {
		processor, err := NewOpenAPIProcessor(cleanFile, "", true)
		if err != nil {
			return fmt.Errorf("failed to create processor: %w", err)
		}
		if err := processor.WriteDocument(ctx, doc); err != nil {
			return fmt.Errorf("failed to write fixed document: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Applied %d fix(es) to %s\n", len(result.Applied), cleanFile)
	}

	return nil
}

func reportFixResults(result *fix.Result, dryRun bool) {
	prefix := ""
	if dryRun {
		prefix = "[dry-run] "
	}

	if len(result.Applied) > 0 {
		fmt.Fprintf(os.Stderr, "\n%sFixed:\n", prefix)
		for _, af := range result.Applied {
			fmt.Fprintf(os.Stderr, "  [%d:%d] %s - %s\n",
				af.Error.GetLineNumber(), af.Error.GetColumnNumber(),
				af.Error.Rule, af.Fix.Description())
			if af.Before != "" || af.After != "" {
				fmt.Fprintf(os.Stderr, "    %s -> %s\n", af.Before, af.After)
			}
		}
	}

	if len(result.Skipped) > 0 {
		fmt.Fprintf(os.Stderr, "\n%sSkipped:\n", prefix)
		for _, sf := range result.Skipped {
			fmt.Fprintf(os.Stderr, "  [%d:%d] %s - %s (%s)\n",
				sf.Error.GetLineNumber(), sf.Error.GetColumnNumber(),
				sf.Error.Rule, sf.Fix.Description(), skipReasonString(sf.Reason))
		}
	}

	if len(result.Failed) > 0 {
		fmt.Fprintf(os.Stderr, "\n%sFailed:\n", prefix)
		for _, ff := range result.Failed {
			fmt.Fprintf(os.Stderr, "  [%d:%d] %s - %s: %v\n",
				ff.Error.GetLineNumber(), ff.Error.GetColumnNumber(),
				ff.Error.Rule, ff.Fix.Description(), ff.FixError)
		}
	}
}

func skipReasonString(reason fix.SkipReason) string {
	switch reason {
	case fix.SkipInteractive:
		return "requires interactive input"
	case fix.SkipConflict:
		return "conflict with previous fix"
	case fix.SkipUser:
		return "skipped by user"
	default:
		return "unknown"
	}
}

// lazyPrompter defers TerminalPrompter creation until an interactive fix is
// actually encountered, avoiding unnecessary setup when all fixes are non-interactive.
type lazyPrompter struct {
	once     sync.Once
	prompter *fix.TerminalPrompter
}

func (l *lazyPrompter) init() {
	l.once.Do(func() {
		l.prompter = fix.NewTerminalPrompter(os.Stdin, os.Stderr)
	})
}

func (l *lazyPrompter) PromptFix(finding *validation.Error, f validation.Fix) ([]string, error) {
	l.init()
	return l.prompter.PromptFix(finding, f)
}

func (l *lazyPrompter) Confirm(message string) (bool, error) {
	l.init()
	return l.prompter.Confirm(message)
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
