package arazzo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/cmd/openapi/commands/cmdutil"
	"github.com/speakeasy-api/openapi/cmd/openapi/internal/arazzoexplore"
	"github.com/spf13/cobra"
)

// maxExploreValidationErrors is the number of validation errors listed before the explorer starts.
const maxExploreValidationErrors = 5

var exploreCmd = &cobra.Command{
	Use:   "explore <input-file>",
	Short: "Interactively explore the workflows of an Arazzo document",
	Long: `Launch an interactive terminal UI to browse the workflows of an Arazzo document.

Each workflow is listed with its steps underneath. Expand a workflow to see its summary,
description, dependsOn, inputs, parameters, success and failure actions and outputs. Expand a step
to see what it targets, its dependsOn, parameters, request body, success criteria, the success and
failure actions that apply to it (including the ones it inherits from its workflow) and its outputs.
Reusable parameters and actions are shown resolved from components.

Stdin is supported — either pipe data directly or use '-' explicitly:
  cat workflow.arazzo.yaml | openapi arazzo explore
  cat workflow.arazzo.yaml | openapi arazzo explore -

Navigation:
  ↑/k           Move up
  ↓/j           Move down
  gg            Jump to top
  G             Jump to bottom
  Ctrl-U        Scroll up by half a screen
  Ctrl-D        Scroll down by half a screen
  [ / ]         Jump to the previous / next workflow
  Enter/Space   Toggle details
  e / E         Expand / collapse all details
  ?             Show help
  q/Ctrl+C      Quit`,
	Args: cmdutil.StdinOrFileArgs(1, 1),
	RunE: runExplore,
}

// exploreIO are the streams used by the explore command.
type exploreIO struct {
	stdin  io.Reader
	stderr io.Writer
}

func runExplore(cmd *cobra.Command, args []string) error {
	m, err := newExploreModel(cmd.Context(), cmdutil.InputFileFromArgs(args), exploreIO{stdin: os.Stdin, stderr: os.Stderr})
	if err != nil {
		return err
	}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		return fmt.Errorf("error running explorer: %w", err)
	}
	return nil
}

// newExploreModel loads an Arazzo document and creates the explorer for it. Validation errors are reported but don't
// stop the document being explored.
func newExploreModel(ctx context.Context, inputFile string, streams exploreIO) (arazzoexplore.Model, error) {
	var data []byte
	var err error
	if cmdutil.IsStdin(inputFile) {
		fmt.Fprintf(streams.stderr, "Note: reading document from stdin; interactive TUI requires /dev/tty for keyboard input.\n")
		data, err = io.ReadAll(streams.stdin)
		if err != nil {
			return arazzoexplore.Model{}, fmt.Errorf("failed to read stdin: %w", err)
		}
	} else {
		data, err = os.ReadFile(filepath.Clean(inputFile))
		if err != nil {
			return arazzoexplore.Model{}, fmt.Errorf("failed to open input file: %w", err)
		}
	}

	doc, validationErrs, err := arazzo.Unmarshal(ctx, bytes.NewReader(data))
	if err != nil {
		return arazzoexplore.Model{}, fmt.Errorf("failed to unmarshal Arazzo document: %w", err)
	}
	if doc == nil {
		return arazzoexplore.Model{}, errors.New("failed to parse Arazzo document: document is nil")
	}

	if len(validationErrs) > 0 {
		fmt.Fprintf(streams.stderr, "⚠️  Found %d validation errors in document:\n", len(validationErrs))
		for i, validationErr := range validationErrs[:min(len(validationErrs), maxExploreValidationErrors)] {
			fmt.Fprintf(streams.stderr, "  %d. %s\n", i+1, validationErr.Error())
		}
		if len(validationErrs) > maxExploreValidationErrors {
			fmt.Fprintf(streams.stderr, "  ... and %d more\n", len(validationErrs)-maxExploreValidationErrors)
		}
		fmt.Fprintln(streams.stderr)
	}

	items := arazzoexplore.Collect(doc)
	if len(items) == 0 {
		return arazzoexplore.Model{}, errors.New("no workflows found in the Arazzo document")
	}

	title := doc.Info.Title
	if title == "" {
		title = "Arazzo"
	}
	version := doc.Info.Version
	if version == "" {
		version = "unknown"
	}

	return arazzoexplore.NewModel(items, title, version, doc.Arazzo), nil
}

// GetExploreCommand returns the explore command for external use
func GetExploreCommand() *cobra.Command {
	return exploreCmd
}
