package openapi

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/speakeasy-api/openapi/cmd/openapi/internal/explore"
	"github.com/speakeasy-api/openapi/cmd/openapi/internal/explore/tui"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/spf13/cobra"
)

var (
	snipWriteInPlace     bool
	snipOperationIDs     []string
	snipOperations       []string
	snipKeepOperationIDs []string
	snipKeepOperations   []string
)

var snipCmd = &cobra.Command{
	Use:   "snip <input-file> [output-file]",
	Short: "Remove operations from an OpenAPI specification",
	Long: `Remove selected operations from an OpenAPI specification and clean up unused components.

Stdin is supported in CLI mode — either pipe data directly or use '-' explicitly:
  cat spec.yaml | openapi spec snip --operationId deleteUser
  cat spec.yaml | openapi spec snip --operationId deleteUser -

This command can operate in two modes:

1. Interactive Mode (no flags specified):
   Launch a terminal UI to browse and select operations for removal.
   - Navigate with j/k or arrow keys
   - Press Space to select/deselect operations
   - Press 'a' to select all, 'A' to deselect all  
   - Press 'w' to write the result
   - Press 'q' or Esc to cancel

2. Command-Line Mode (--operationId or --operation flags):
   Remove operations specified via flags without launching the UI.

Output options:
- No output file: writes to stdout (pipe-friendly)
- Output file specified: writes to the specified file
- --write flag: writes in-place to the input file

Examples:

  # Interactive mode - browse and select operations
  openapi spec snip ./spec.yaml
  openapi spec snip ./spec.yaml ./snipped-spec.yaml
  openapi spec snip -w ./spec.yaml

  # CLI mode - remove by operation ID (multiple flags)
  openapi spec snip --operationId deleteUser --operationId adminDebug ./spec.yaml

  # CLI mode - remove by operation ID (comma-separated)
  openapi spec snip --operationId deleteUser,adminDebug ./spec.yaml

  # CLI mode - remove by path:method (multiple flags)
  openapi spec snip --operation /users/{id}:DELETE --operation /admin:GET ./spec.yaml

  # CLI mode - remove by path:method (comma-separated)
  openapi spec snip --operation /users/{id}:DELETE,/admin:GET ./spec.yaml

  # CLI mode - mixed operation IDs and path:method
  openapi spec snip --operationId deleteUser --operation /admin:GET ./spec.yaml

  # CLI mode - write to stdout for piping
  openapi spec snip --operation /internal/debug:GET ./spec.yaml > ./public-spec.yaml`,
	Args: stdinOrFileArgs(1, 2),
	RunE: runSnip,
}

func init() {
	snipCmd.Flags().BoolVarP(&snipWriteInPlace, "write", "w", false, "write result in-place to input file")
	snipCmd.Flags().StringSliceVar(&snipOperationIDs, "operationId", nil, "operation ID to remove (can be comma-separated or repeated)")
	snipCmd.Flags().StringSliceVar(&snipOperations, "operation", nil, "operation as path:method to remove (can be comma-separated or repeated)")
	// Keep-mode flags (mutually exclusive with remove-mode flags)
	snipCmd.Flags().StringSliceVar(&snipKeepOperationIDs, "keepOperationId", nil, "operation ID to keep (can be comma-separated or repeated)")
	snipCmd.Flags().StringSliceVar(&snipKeepOperations, "keepOperation", nil, "operation as path:method to keep (can be comma-separated or repeated)")
}

func runSnip(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	inputFile := inputFileFromArgs(args)

	outputFile := outputFileFromArgs(args)

	// Check which flag sets were specified
	hasRemoveFlags := len(snipOperationIDs) > 0 || len(snipOperations) > 0
	hasKeepFlags := len(snipKeepOperationIDs) > 0 || len(snipKeepOperations) > 0

	// If -w is specified without any operation selection flags, error
	if snipWriteInPlace && !(hasRemoveFlags || hasKeepFlags) {
		return errors.New("--write flag requires specifying operations via --operationId/--operation or --keepOperationId/--keepOperation")
	}

	// Interactive mode when no flags provided (not supported with stdin)
	if !hasRemoveFlags && !hasKeepFlags {
		if IsStdin(inputFile) {
			return errors.New("interactive mode is not supported when reading from stdin; use --operationId or --operation flags")
		}
		return runSnipInteractive(ctx, inputFile, outputFile)
	}

	// Disallow mixing keep + remove flags; ambiguous intent
	if hasRemoveFlags && hasKeepFlags {
		return errors.New("cannot combine keep and remove flags; use either --operationId/--operation or --keepOperationId/--keepOperation")
	}

	// CLI mode
	if hasKeepFlags {
		return runSnipCLIKeep(ctx, inputFile, outputFile)
	}
	return runSnipCLI(ctx, inputFile, outputFile)
}

func runSnipCLI(ctx context.Context, inputFile, outputFile string) error {
	// Create processor
	processor, err := NewOpenAPIProcessor(inputFile, outputFile, snipWriteInPlace)
	if err != nil {
		return err
	}

	// Load document
	doc, validationErrors, err := processor.LoadDocument(ctx)
	if err != nil {
		return err
	}

	// Report validation errors (if any)
	processor.ReportValidationErrors(validationErrors)

	// Parse operation flags
	operationsToRemove, err := parseOperationFlags()
	if err != nil {
		return err
	}

	if len(operationsToRemove) == 0 {
		return errors.New("no operations specified for removal")
	}

	// Perform the snip
	removed, err := openapi.Snip(ctx, doc, operationsToRemove)
	if err != nil {
		return fmt.Errorf("failed to snip operations: %w", err)
	}

	processor.PrintSuccess(fmt.Sprintf("Successfully removed %d operation(s) and cleaned unused components", removed))

	// Write the snipped document
	return processor.WriteDocument(ctx, doc)
}

func runSnipCLIKeep(ctx context.Context, inputFile, outputFile string) error {
	// Create processor
	processor, err := NewOpenAPIProcessor(inputFile, outputFile, snipWriteInPlace)
	if err != nil {
		return err
	}

	// Load document
	doc, validationErrors, err := processor.LoadDocument(ctx)
	if err != nil {
		return err
	}

	// Report validation errors (if any)
	processor.ReportValidationErrors(validationErrors)

	// Parse keep flags
	keepOps, err := parseKeepOperationFlags()
	if err != nil {
		return err
	}
	if len(keepOps) == 0 {
		return errors.New("no operations specified to keep")
	}

	// Collect all operations from the document
	allOps, err := explore.CollectOperations(ctx, doc)
	if err != nil {
		return fmt.Errorf("failed to collect operations: %w", err)
	}
	if len(allOps) == 0 {
		return errors.New("no operations found in the OpenAPI document")
	}

	// Build lookup sets for keep filters
	keepByID := map[string]bool{}
	keepByPathMethod := map[string]bool{}
	for _, k := range keepOps {
		if k.OperationID != "" {
			keepByID[k.OperationID] = true
		}
		if k.Path != "" && k.Method != "" {
			key := strings.ToUpper(k.Method) + " " + k.Path
			keepByPathMethod[key] = true
		}
	}

	// Compute removal list = all - keep
	var operationsToRemove []openapi.OperationIdentifier
	for _, op := range allOps {
		if op.OperationID != "" && keepByID[op.OperationID] {
			continue
		}
		key := strings.ToUpper(op.Method) + " " + op.Path
		if keepByPathMethod[key] {
			continue
		}
		operationsToRemove = append(operationsToRemove, openapi.OperationIdentifier{
			Path:   op.Path,
			Method: strings.ToUpper(op.Method),
		})
	}

	// If nothing to remove, write as-is
	if len(operationsToRemove) == 0 {
		processor.PrintSuccess("No operations to remove based on keep filters; writing document unchanged")
		return processor.WriteDocument(ctx, doc)
	}

	// Perform the snip
	removed, err := openapi.Snip(ctx, doc, operationsToRemove)
	if err != nil {
		return fmt.Errorf("failed to snip operations: %w", err)
	}

	processor.PrintSuccess(fmt.Sprintf("Successfully kept %d operation(s) and removed %d operation(s) with cleanup", len(allOps)-removed, removed))

	// Write the snipped document
	return processor.WriteDocument(ctx, doc)
}

func runSnipInteractive(ctx context.Context, inputFile, outputFile string) error {
	// Load the OpenAPI document
	doc, err := loadOpenAPIDocument(ctx, inputFile)
	if err != nil {
		return err
	}

	// Collect operations
	operations, err := explore.CollectOperations(ctx, doc)
	if err != nil {
		return fmt.Errorf("failed to collect operations: %w", err)
	}

	if len(operations) == 0 {
		return errors.New("no operations found in the OpenAPI document")
	}

	// Get document info
	docTitle := doc.Info.Title
	if docTitle == "" {
		docTitle = "OpenAPI"
	}
	docVersion := doc.Info.Version
	if docVersion == "" {
		docVersion = "unknown"
	}

	// Create TUI config for snip mode
	exploreConfig := tui.ExploreConfig{
		Title:          "OpenAPI Spec Snip - Select Operations to Remove",
		ModeLabel:      "Snip Mode",
		FooterHelpText: "Space: select | a: all | A: none | w: write | ?: help | q: cancel",
		HelpTitle:      "Snip Mode Help",
	}

	selectionConfig := tui.SelectionConfig{
		Enabled:      true,
		SelectIcon:   "✂️",
		SelectColor:  "#10B981", // colorGreen
		StatusFormat: "Selected: %d operations",
		ActionKeys: []tui.ActionKey{
			{Key: "w", Label: "Write and save"},
		},
	}

	config := tui.Config{
		Explore:   exploreConfig,
		Selection: selectionConfig,
	}

	// Create and run the TUI
	m := tui.NewModelWithConfig(operations, docTitle, docVersion, config)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running snip UI: %w", err)
	}

	// Get the final model state
	tuiModel, ok := finalModel.(tui.Model)
	if !ok {
		return errors.New("unexpected model type")
	}

	// Check if user performed an action or just quit
	actionKey := tuiModel.GetActionKey()
	if actionKey == "" {
		// User cancelled (pressed q or Esc)
		fmt.Fprintln(os.Stderr, "Snip cancelled - no changes made")
		return nil
	}

	// Get selected operations
	selectedOps := tuiModel.GetSelectedOperations()
	if len(selectedOps) == 0 {
		fmt.Fprintln(os.Stderr, "No operations selected - no changes made")
		return nil
	}

	// Convert to operation identifiers
	var operationsToRemove []openapi.OperationIdentifier
	for _, op := range selectedOps {
		operationsToRemove = append(operationsToRemove, openapi.OperationIdentifier{
			Path:   op.Path,
			Method: op.Method,
		})
	}

	// Perform the snip
	removed, err := openapi.Snip(ctx, doc, operationsToRemove)
	if err != nil {
		return fmt.Errorf("failed to snip operations: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✅ Successfully removed %d operation(s) and cleaned unused components\n", removed)

	// Determine default output path (prefer outputFile if specified, otherwise inputFile)
	defaultPath := outputFile
	if defaultPath == "" {
		defaultPath = inputFile
	}

	// Prompt user for output location using TUI
	finalOutputFile, err := tui.PromptForFilePath("Save snipped spec to:", defaultPath)
	if err != nil {
		return fmt.Errorf("error prompting for file path: %w", err)
	}

	if finalOutputFile == "" {
		// User cancelled
		fmt.Fprintln(os.Stderr, "Cancelled - no changes saved")
		return nil
	}

	// Write the result
	writeInPlace := (finalOutputFile == inputFile)
	processor, err := NewOpenAPIProcessor(inputFile, finalOutputFile, writeInPlace)
	if err != nil {
		return err
	}

	return processor.WriteDocument(ctx, doc)
}

// parseOperationFlags parses the operation flags into operation identifiers
// Handles both repeated flags and comma-separated values
func parseOperationFlags() ([]openapi.OperationIdentifier, error) {
	var operations []openapi.OperationIdentifier

	// Parse operation IDs (handles comma-separated values automatically via StringSlice)
	for _, opID := range snipOperationIDs {
		if opID != "" {
			operations = append(operations, openapi.OperationIdentifier{
				OperationID: opID,
			})
		}
	}

	// Parse path:method operations (handles comma-separated values automatically)
	for _, op := range snipOperations {
		if op == "" {
			continue
		}

		// Must be in path:method format
		parts := strings.SplitN(op, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid operation format: %s (expected path:METHOD format, e.g., /users:GET)", op)
		}

		path := parts[0]
		method := strings.ToUpper(parts[1])

		if path == "" || method == "" {
			return nil, fmt.Errorf("invalid operation format: %s (path and method cannot be empty)", op)
		}

		operations = append(operations, openapi.OperationIdentifier{
			Path:   path,
			Method: method,
		})
	}

	return operations, nil
}

// parseKeepOperationFlags parses the keep flags into operation identifiers
// Handles both repeated flags and comma-separated values
func parseKeepOperationFlags() ([]openapi.OperationIdentifier, error) {
	var operations []openapi.OperationIdentifier

	// Parse keep operation IDs
	for _, opID := range snipKeepOperationIDs {
		if opID != "" {
			operations = append(operations, openapi.OperationIdentifier{
				OperationID: opID,
			})
		}
	}

	// Parse keep path:method operations
	for _, op := range snipKeepOperations {
		if op == "" {
			continue
		}

		parts := strings.SplitN(op, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid keep operation format: %s (expected path:METHOD format, e.g., /users:GET)", op)
		}

		path := parts[0]
		method := strings.ToUpper(parts[1])

		if path == "" || method == "" {
			return nil, fmt.Errorf("invalid keep operation format: %s (path and method cannot be empty)", op)
		}

		operations = append(operations, openapi.OperationIdentifier{
			Path:   path,
			Method: method,
		})
	}

	return operations, nil
}

// GetSnipCommand returns the snip command for external use
func GetSnipCommand() *cobra.Command {
	return snipCmd
}
