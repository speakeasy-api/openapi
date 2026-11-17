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
	arazzoLinter "github.com/speakeasy-api/openapi/arazzo/linter"
	"github.com/speakeasy-api/openapi/cmd/openapi/commands/cmdutil"
	"github.com/speakeasy-api/openapi/linter"
	"github.com/speakeasy-api/openapi/linter/fix"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/spf13/cobra"
)

var lintCmd = &cobra.Command{
	Use:   "lint <file>",
	Short: "Lint an Arazzo workflow document",
	Long: `Lint an Arazzo workflow document for semantic issues, style and security concerns.

This command reports everything ` + "`openapi arazzo validate`" + ` does, plus lint rules such as:
- Steps that can never run, and workflows or steps that depend on each other in a cycle
- Unused components, workflow inputs and step outputs
- Steps without successCriteria and retries without retryAfter
- Source descriptions without a type or loaded over plain HTTP

By default the documents referenced by sourceDescriptions are loaded and the workflows are checked
against them, as with ` + "`openapi arazzo validate`" + `. Use --skip-sources to only lint the document itself.

Stdin is supported — either pipe data directly or use '-' explicitly:
  cat workflow.yaml | openapi arazzo lint
  cat workflow.yaml | openapi arazzo lint -

CONFIGURATION:

By default, the linter looks for a configuration file at ~/.openapi/arazzo-lint.yaml (separate from the
OpenAPI linter's configuration, as the rules and rulesets differ). Use --config to specify a configuration file.

Available rulesets: all (default), recommended

Example configuration (arazzo-lint.yaml):

  extends: recommended

  rules:
    - id: style-step-success-criteria
      severity: error
    - id: semantic-unused-step-output
      disabled: true

Use ` + "`openapi arazzo list-rules`" + ` to see all rules.

AUTOFIXING:

Use --fix to apply fixes that don't need input, such as removing unused components, or setting a
sourceDescription's type from the document it references. Use --fix-interactive to also be prompted for
fixes that need input. Use --dry-run with either flag to preview the fixes without changing the file.
Fixes are not supported when reading from stdin.`,
	Args:    cmdutil.StdinOrFileArgs(1, 1),
	PreRunE: validateArazzoLintFlags,
	Run:     runArazzoLint,
}

// lintFlags are the flags of the lint command.
type lintFlags struct {
	format         string
	ruleset        string
	configFile     string
	disableRules   []string
	summary        bool
	skipSources    bool
	fix            bool
	fixInteractive bool
	dryRun         bool
}

// lintIO are the streams used by the lint command.
type lintIO struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

var arazzoLintFlags lintFlags

func init() {
	lintCmd.Flags().StringVarP(&arazzoLintFlags.format, "format", "f", "text", "Output format: text or json")
	lintCmd.Flags().StringVarP(&arazzoLintFlags.ruleset, "ruleset", "r", "", "Ruleset to use: all or recommended (default from config, or all)")
	lintCmd.Flags().StringVarP(&arazzoLintFlags.configFile, "config", "c", "", "Path to lint config file (default: ~/.openapi/arazzo-lint.yaml)")
	lintCmd.Flags().StringSliceVarP(&arazzoLintFlags.disableRules, "disable", "d", nil, "Rule IDs to disable (can be repeated)")
	lintCmd.Flags().BoolVar(&arazzoLintFlags.summary, "summary", false, "Print a per-rule summary table of findings")
	lintCmd.Flags().BoolVar(&arazzoLintFlags.skipSources, "skip-sources", false, "don't load source descriptions to check the document against them")
	lintCmd.Flags().BoolVar(&arazzoLintFlags.fix, "fix", false, "Automatically apply non-interactive fixes and write back")
	lintCmd.Flags().BoolVar(&arazzoLintFlags.fixInteractive, "fix-interactive", false, "Apply all fixes, prompting for interactive ones")
	lintCmd.Flags().BoolVar(&arazzoLintFlags.dryRun, "dry-run", false, "Show what fixes would be applied without changing the file (requires --fix or --fix-interactive)")
}

func validateArazzoLintFlags(_ *cobra.Command, args []string) error {
	return arazzoLintFlags.validate(cmdutil.InputFileFromArgs(args))
}

func (f lintFlags) validate(file string) error {
	if f.fix && f.fixInteractive {
		return errors.New("--fix and --fix-interactive are mutually exclusive")
	}
	if f.dryRun && !f.fix && !f.fixInteractive {
		return errors.New("--dry-run requires --fix or --fix-interactive")
	}
	if cmdutil.IsStdin(file) && (f.fix || f.fixInteractive) {
		return errors.New("--fix and --fix-interactive are not supported when reading from stdin")
	}
	return nil
}

func runArazzoLint(cmd *cobra.Command, args []string) {
	streams := lintIO{stdin: os.Stdin, stdout: os.Stdout, stderr: os.Stderr}
	if err := lintArazzo(cmd.Context(), cmdutil.InputFileFromArgs(args), arazzoLintFlags, streams); err != nil {
		cmdutil.Die(err)
	}
}

func lintArazzo(ctx context.Context, file string, flags lintFlags, streams lintIO) error {
	if err := flags.validate(file); err != nil {
		return err
	}

	config, err := loadArazzoLintConfig(flags)
	if err != nil {
		return err
	}

	displayFile := "stdin"
	location := ""
	var data []byte
	if cmdutil.IsStdin(file) {
		fmt.Fprintf(streams.stderr, "Linting Arazzo document from stdin\n")
		data, err = io.ReadAll(streams.stdin)
		if err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}
	} else {
		location = filepath.Clean(file)
		displayFile = location
		fmt.Fprintf(streams.stderr, "Linting Arazzo document: %s\n", location)
		data, err = os.ReadFile(location)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
	}

	lint := arazzoLinter.NewLinter(config)
	doc, output, err := lintArazzoDocument(ctx, lint, data, location, flags.skipSources)
	if err != nil {
		return err
	}

	if flags.fix || flags.fixInteractive {
		applied, err := applyArazzoFixes(ctx, flags, doc, output, location, streams)
		if err != nil {
			return err
		}
		// Re-lint the fixed document so the remaining findings are accurate.
		if applied {
			data, err = os.ReadFile(location)
			if err != nil {
				return fmt.Errorf("failed to reopen file after fix: %w", err)
			}
			if _, output, err = lintArazzoDocument(ctx, lint, data, location, flags.skipSources); err != nil {
				return err
			}
		}
	}

	if flags.format == "json" {
		fmt.Fprintln(streams.stdout, output.FormatJSON())
	} else {
		fmt.Fprintln(streams.stdout, displayFile)
		fmt.Fprintln(streams.stdout, output.FormatText())
	}
	if flags.summary {
		fmt.Fprintln(streams.stdout, output.FormatSummary())
	}

	if output.HasErrors() {
		return fmt.Errorf("linting found %d errors", output.ErrorCount())
	}
	return nil
}

// lintArazzoDocument parses, validates (including against the document's sources unless skipped) and lints the
// document.
func lintArazzoDocument(ctx context.Context, lint *arazzoLinter.Linter, data []byte, location string, skipSources bool) (*arazzo.Arazzo, *linter.Output, error) {
	doc, validationErrs, err := arazzo.Unmarshal(ctx, bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal file: %w", err)
	}

	if !skipSources {
		sources, loadErrs := arazzo.LoadSourceDescriptions(ctx, doc, arazzo.LoadSourcesOptions{DocumentLocation: location})
		validationErrs = append(validationErrs, loadErrs...)
		validationErrs = append(validationErrs, doc.ValidateSources(ctx, sources)...)
		ctx = arazzoLinter.WithSourceDocuments(ctx, sources)
	}

	output, err := lint.Lint(ctx, linter.NewDocumentInfo(doc, location), validationErrs, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("linting failed: %w", err)
	}
	return doc, output, nil
}

// applyArazzoFixes applies the fixes for the lint findings and writes the document back, returning whether the
// file was changed.
func applyArazzoFixes(ctx context.Context, flags lintFlags, doc *arazzo.Arazzo, output *linter.Output, location string, streams lintIO) (bool, error) {
	opts := fix.Options{Mode: fix.ModeAuto, DryRun: flags.dryRun}
	var prompter validation.Prompter
	if flags.fixInteractive {
		opts.Mode = fix.ModeInteractive
		prompter = fix.NewTerminalPrompter(streams.stdin, streams.stderr)
	}

	result, err := fix.NewEngine(opts, prompter, fix.NewFixRegistry()).ProcessDocumentErrors(ctx, doc, output.Results)
	if err != nil {
		return false, fmt.Errorf("fix processing failed: %w", err)
	}
	reportArazzoFixResults(streams.stderr, result, flags.dryRun)

	if len(result.Applied) == 0 || flags.dryRun {
		return false, nil
	}

	var buf bytes.Buffer
	if err := arazzo.Marshal(ctx, doc, &buf); err != nil {
		return false, fmt.Errorf("failed to marshal fixed document: %w", err)
	}
	if err := os.WriteFile(location, buf.Bytes(), 0o644); err != nil { //nolint:gosec // the input file is user specified
		return false, fmt.Errorf("failed to write fixed document: %w", err)
	}
	fmt.Fprintf(streams.stderr, "Applied %d fix(es) to %s\n", len(result.Applied), location)
	return true, nil
}

func reportArazzoFixResults(w io.Writer, result *fix.Result, dryRun bool) {
	prefix := ""
	if dryRun {
		prefix = "[dry-run] "
	}

	if len(result.Applied) > 0 {
		fmt.Fprintf(w, "\n%sFixed:\n", prefix)
		for _, af := range result.Applied {
			fmt.Fprintf(w, "  [%d:%d] %s - %s\n", af.Error.GetLineNumber(), af.Error.GetColumnNumber(), af.Error.Rule, af.Fix.Description())
		}
	}
	if len(result.Skipped) > 0 {
		fmt.Fprintf(w, "\n%sSkipped:\n", prefix)
		for _, sf := range result.Skipped {
			fmt.Fprintf(w, "  [%d:%d] %s - %s\n", sf.Error.GetLineNumber(), sf.Error.GetColumnNumber(), sf.Error.Rule, sf.Fix.Description())
		}
	}
	if len(result.Failed) > 0 {
		fmt.Fprintf(w, "\n%sFailed:\n", prefix)
		for _, ff := range result.Failed {
			fmt.Fprintf(w, "  [%d:%d] %s - %s: %v\n", ff.Error.GetLineNumber(), ff.Error.GetColumnNumber(), ff.Error.Rule, ff.Fix.Description(), ff.FixError)
		}
	}
}

// loadArazzoLintConfig loads the lint configuration from --config, or ~/.openapi/arazzo-lint.yaml if it exists,
// and applies the command line overrides.
func loadArazzoLintConfig(flags lintFlags) (*linter.Config, error) {
	config := linter.NewConfig()

	if flags.configFile != "" {
		loaded, err := linter.LoadConfigFromFile(flags.configFile)
		if err != nil {
			return nil, err
		}
		config = loaded
	} else if homeDir, err := os.UserHomeDir(); err == nil {
		if loaded, err := linter.LoadConfigFromFile(filepath.Join(homeDir, ".openapi", "arazzo-lint.yaml")); err == nil {
			config = loaded
		}
	}

	if flags.ruleset != "" {
		if _, ok := arazzoLinter.NewLinter(nil).Registry().GetRuleset(flags.ruleset); !ok {
			return nil, fmt.Errorf("unknown ruleset `%s`, available rulesets: all, recommended", flags.ruleset)
		}
		config.Extends = []string{flags.ruleset}
	}

	for _, rule := range flags.disableRules {
		disabled := true
		config.Rules = append(config.Rules, linter.RuleEntry{ID: rule, Disabled: &disabled})
	}

	config.OutputFormat = linter.OutputFormatText
	if flags.format == "json" {
		config.OutputFormat = linter.OutputFormatJSON
	}

	return config, nil
}

// GetLintCommand returns the lint command for external use
func GetLintCommand() *cobra.Command {
	return lintCmd
}
