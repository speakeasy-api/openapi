package openapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/speakeasy-api/openapi/cmd/openapi/internal/explore"
	"github.com/speakeasy-api/openapi/cmd/openapi/internal/explore/tui"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/spf13/cobra"
)

var exploreCmd = &cobra.Command{
	Use:   "explore <file>",
	Short: "Interactively explore an OpenAPI specification",
	Long: `Launch an interactive terminal UI to browse and explore OpenAPI operations.

Use '-' as the file argument to read from stdin:
  cat spec.yaml | openapi spec explore -

This command provides a user-friendly interface for navigating through API
endpoints, viewing operation details, parameters, request/response information,
and more.

Navigation:
  ↑/k           Move up
  ↓/j           Move down
  gg            Jump to top
  G             Jump to bottom
  Ctrl-U        Scroll up by half a screen
  Ctrl-D        Scroll down by half a screen
  Enter/Space   Toggle operation details
  ?             Show help
  q/Esc         Quit

The explore command helps you understand API structure and operation details
without needing to manually parse the OpenAPI specification file.`,
	Args: cobra.ExactArgs(1),
	RunE: runExplore,
}

func runExplore(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	inputFile := args[0]

	// Load the OpenAPI document
	doc, err := loadOpenAPIDocument(ctx, inputFile)
	if err != nil {
		return err
	}

	// Collect operations from the document
	operations, err := explore.CollectOperations(ctx, doc)
	if err != nil {
		return fmt.Errorf("failed to collect operations: %w", err)
	}

	if len(operations) == 0 {
		return errors.New("no operations found in the OpenAPI document")
	}

	// Get document info for display
	docTitle := doc.Info.Title
	if docTitle == "" {
		docTitle = "OpenAPI"
	}
	docVersion := doc.Info.Version
	if docVersion == "" {
		docVersion = "unknown"
	}

	// Create and run the TUI
	m := tui.NewModel(operations, docTitle, docVersion)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running explorer: %w", err)
	}

	return nil
}

// loadOpenAPIDocument loads an OpenAPI document from a file or stdin (using "-").
func loadOpenAPIDocument(ctx context.Context, file string) (*openapi.OpenAPI, error) {
	var reader io.ReadCloser

	if IsStdin(file) {
		reader = io.NopCloser(os.Stdin)
	} else {
		cleanFile := filepath.Clean(file)
		f, err := os.Open(cleanFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %w", err)
		}
		reader = f
	}
	defer reader.Close()

	doc, validationErrors, err := openapi.Unmarshal(ctx, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal OpenAPI document: %w", err)
	}
	if doc == nil {
		return nil, errors.New("failed to parse OpenAPI document: document is nil")
	}

	// Report validation errors as warnings but continue
	if len(validationErrors) > 0 {
		fmt.Fprintf(os.Stderr, "⚠️  Found %d validation errors in document:\n", len(validationErrors))
		for i, validationErr := range validationErrors {
			if i < 5 { // Limit to first 5 errors
				fmt.Fprintf(os.Stderr, "  %d. %s\n", i+1, validationErr.Error())
			}
		}
		if len(validationErrors) > 5 {
			fmt.Fprintf(os.Stderr, "  ... and %d more\n", len(validationErrors)-5)
		}
		fmt.Fprintln(os.Stderr)
	}

	return doc, nil
}

// GetExploreCommand returns the explore command for external use
func GetExploreCommand() *cobra.Command {
	return exploreCmd
}
