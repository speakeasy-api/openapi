package openapi

import (
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/speakeasy-api/openapi/cmd/openapi/internal/analyze"
	"github.com/speakeasy-api/openapi/cmd/openapi/internal/analyze/tui"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze <file>",
	Short: "Analyze schema complexity, cyclicality, and codegen difficulty",
	Long: `Analyze an OpenAPI specification to understand schema complexity.

This command examines schema references to identify:
- Cycles and strongly connected components (SCCs)
- Per-schema complexity metrics (fan-in, fan-out, nesting)
- Code generation difficulty tiers (green/yellow/red)
- Actionable refactoring suggestions

Output formats:
  tui   - Interactive terminal UI with progressive disclosure (default)
  json  - Machine-readable JSON report for CI/CD pipelines
  text  - Human-readable text summary
  dot   - Graphviz DOT format for graph visualization

Stdin is supported — pipe data or use '-':
  cat spec.yaml | openapi spec analyze
  cat spec.yaml | openapi spec analyze - --format json`,
	Args: stdinOrFileArgs(1, 1),
	RunE: runAnalyze,
}

func init() {
	analyzeCmd.Flags().StringP("format", "f", "tui", "output format: tui, json, text, dot")
	analyzeCmd.Flags().StringP("output", "o", "", "write output to file instead of stdout")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	inputFile := inputFileFromArgs(args)
	format, _ := cmd.Flags().GetString("format")
	outputFile, _ := cmd.Flags().GetString("output")

	// Load the document
	doc, err := loadOpenAPIDocument(ctx, inputFile)
	if err != nil {
		return err
	}

	// Run analysis
	report := analyze.Analyze(ctx, doc)

	switch format {
	case "tui":
		if outputFile != "" {
			return errors.New("--output is not compatible with --format tui")
		}
		m := tui.NewModel(report)
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running analyzer TUI: %w", err)
		}
		return nil

	case "json":
		w := os.Stdout
		if outputFile != "" {
			f, err := os.Create(outputFile)
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}
			defer f.Close()
			w = f
		}
		return analyze.WriteJSON(w, report)

	case "text":
		w := os.Stdout
		if outputFile != "" {
			f, err := os.Create(outputFile)
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}
			defer f.Close()
			w = f
		}
		analyze.WriteText(w, report)
		return nil

	case "dot":
		w := os.Stdout
		if outputFile != "" {
			f, err := os.Create(outputFile)
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}
			defer f.Close()
			w = f
		}
		analyze.WriteDOT(w, report)
		return nil

	default:
		return fmt.Errorf("unknown format: %s (expected tui, json, text, or dot)", format)
	}
}
