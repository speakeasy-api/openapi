package main

import (
	"bytes"
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/speakeasy-api/openapi/cmd/openapi/internal/tui/navigator"
	openapiTUI "github.com/speakeasy-api/openapi/cmd/openapi/internal/tui/openapi"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/spf13/cobra"
)

var viewCmd = &cobra.Command{
	Use:   "view <openapi-file>",
	Short: "Interactively view and explore an OpenAPI document",
	Long: `View and explore an OpenAPI document in an interactive terminal interface.

This command opens an interactive viewer that allows you to navigate through
the OpenAPI document structure using keyboard controls:

- ↑/↓: Navigate up and down through items
- →/Enter: Enter selected item to explore deeper
- ←/Esc: Go back to parent level
- ?: Show help
- q: Quit

The viewer shows a hierarchical view of the OpenAPI document, including:
- Document information (title, version, description)
- Servers
- Paths and operations with their details
- Components (schemas, parameters, responses, etc.)
- Security requirements
- Tags

References ($ref) are shown as navigable items that can be explored
when selected, providing lazy loading of referenced content.`,
	Args: cobra.ExactArgs(1),
	RunE: runView,
}

func runView(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	inputFile := args[0]

	// Load and parse the OpenAPI document
	doc, err := loadOpenAPIDocument(ctx, inputFile)
	if err != nil {
		return fmt.Errorf("failed to load OpenAPI document: %w", err)
	}

	// Build the navigation tree
	root := openapiTUI.BuildTree(doc)

	// Create the explorer
	explorer := navigator.NewExplorer(root)

	// Create and run the bubbletea program
	program := tea.NewProgram(explorer, tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		return fmt.Errorf("failed to run interactive viewer: %w", err)
	}

	return nil
}

// loadOpenAPIDocument loads and parses an OpenAPI document from a file
func loadOpenAPIDocument(ctx context.Context, filePath string) (*openapi.OpenAPI, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse the OpenAPI document
	doc, validationErrs, err := openapi.Unmarshal(ctx, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI document: %w", err)
	}
	// TODO allow viewing of validation errors
	_ = validationErrs

	if doc == nil {
		return nil, fmt.Errorf("parsed document is nil")
	}

	return doc, nil
}
