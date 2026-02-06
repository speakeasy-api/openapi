package openapi

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/speakeasy-api/openapi/openapi/linter/converter"
	"github.com/spf13/cobra"
)

var convertRulesCmd = &cobra.Command{
	Use:   "convert-rules <config-file>",
	Short: "Convert Spectral/Vacuum/legacy configs to native linter format",
	Long: `Convert a Spectral, Vacuum, or legacy Speakeasy lint config into the native
linter format. This generates:

  - A lint.yaml config file with mapped rule overrides
  - TypeScript rule files for custom rules that don't have native equivalents

Supported input formats:
  - Spectral configs (.spectral.yml / .spectral.yaml)
  - Vacuum configs (Spectral-compatible format)
  - Legacy Speakeasy lint.yaml (with lintVersion/defaultRuleset/rulesets)

Examples:
  openapi spec lint convert-rules .spectral.yml
  openapi spec lint convert-rules .spectral.yml --output ./converted
  openapi spec lint convert-rules lint.yaml --dry-run
  openapi spec lint convert-rules .spectral.yml --force`,
	Args: cobra.ExactArgs(1),
	Run:  runConvertRules,
}

var (
	convertOutput   string
	convertRulesDir string
	convertForce    bool
	convertDryRun   bool
)

func init() {
	convertRulesCmd.Flags().StringVarP(&convertOutput, "output", "o", ".", "Output directory for generated files")
	convertRulesCmd.Flags().StringVar(&convertRulesDir, "rules-dir", "./rules", "Subdirectory for generated .ts rule files")
	convertRulesCmd.Flags().BoolVarP(&convertForce, "force", "f", false, "Overwrite existing files")
	convertRulesCmd.Flags().BoolVar(&convertDryRun, "dry-run", false, "Print summary without writing files")

	lintCmd.AddCommand(convertRulesCmd)
}

func runConvertRules(cmd *cobra.Command, args []string) {
	configFile := args[0]

	// Parse the input config
	ir, err := converter.ParseFile(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing config: %v\n", err)
		os.Exit(1)
	}

	// Generate native output
	result, err := converter.Generate(ir,
		converter.WithRulesDir(convertRulesDir),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating output: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	printConvertSummary(result, configFile)

	// Print warnings
	if len(result.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, w := range result.Warnings {
			prefix := ""
			if w.RuleID != "" {
				prefix = fmt.Sprintf("[%s] ", w.RuleID)
			}
			fmt.Printf("  %s(%s) %s\n", prefix, w.Phase, w.Message)
		}
	}

	if convertDryRun {
		fmt.Println("\n--dry-run: no files written")
		return
	}

	// Check for existing files unless --force
	if !convertForce {
		configPath := filepath.Join(convertOutput, "lint.yaml")
		if _, err := os.Stat(configPath); err == nil {
			fmt.Fprintf(os.Stderr, "Error: %s already exists (use --force to overwrite)\n", configPath)
			os.Exit(1)
		}
		rulesPath := filepath.Join(convertOutput, convertRulesDir)
		if _, err := os.Stat(rulesPath); err == nil {
			fmt.Fprintf(os.Stderr, "Error: %s already exists (use --force to overwrite)\n", rulesPath)
			os.Exit(1)
		}
	}

	// Ensure output directory exists
	if err := os.MkdirAll(convertOutput, 0o755); err != nil { //nolint:gosec
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Write files
	if err := result.WriteFiles(convertOutput); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing files: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nFiles written to %s\n", convertOutput)
}

func printConvertSummary(result *converter.GenerateResult, inputFile string) {
	fmt.Printf("Converting: %s\n\n", inputFile)

	// Extends
	if len(result.Config.Extends) > 0 {
		fmt.Printf("Extends: %v\n", result.Config.Extends)
	}

	// Rule overrides
	overrideCount := 0
	for _, entry := range result.Config.Rules {
		if entry.Disabled != nil || entry.Severity != nil {
			overrideCount++
		}
	}
	if overrideCount > 0 {
		fmt.Printf("Rule overrides: %d\n", overrideCount)
	}

	// Generated rules
	if len(result.GeneratedRules) > 0 {
		ruleIDs := sortedKeys(result.GeneratedRules)
		fmt.Printf("Generated rules: %d\n", len(result.GeneratedRules))
		for _, ruleID := range ruleIDs {
			fmt.Printf("  - %s.ts\n", ruleID)
		}

		// Files to be written
		fmt.Println("\nFiles:")
		fmt.Println("  - lint.yaml")
		for _, ruleID := range ruleIDs {
			fmt.Printf("  - %s/%s.ts\n", convertRulesDir, ruleID)
		}
	} else {
		fmt.Println("\nFiles:")
		fmt.Println("  - lint.yaml")
	}
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
