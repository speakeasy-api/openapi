package arazzo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/speakeasy-api/openapi/arazzo"
	"github.com/speakeasy-api/openapi/arazzo/graph"
	"github.com/speakeasy-api/openapi/cmd/openapi/commands/cmdutil"
	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:   "graph <input-file> [output-file]",
	Short: "Render the workflows of an Arazzo document as a Mermaid or Graphviz DOT diagram",
	Long: `Render the workflows of an Arazzo document as a Mermaid flowchart or Graphviz DOT digraph.

Each workflow is drawn as a group containing its steps, showing:
- The order steps run in, following on from one step to the next unless a success action ends the workflow
  or goes elsewhere
- Success action gotos (green) and failure action gotos and retries (dashed red)
- Workflow and step dependsOn entries (dashed grey)
- Steps that run another workflow (thick)

Workflows and steps outside the diagram, such as those in other Arazzo documents referenced by
sourceDescriptions, are drawn with dashed borders.

Stdin is supported — either pipe data directly or use '-' explicitly:
  cat workflow.yaml | openapi arazzo graph
  cat workflow.yaml | openapi arazzo graph -

The format defaults to mermaid, or dot when the output file ends in .dot or .gv.

Examples:
  openapi arazzo graph workflow.arazzo.yaml
  openapi arazzo graph workflow.arazzo.yaml workflow.mmd
  openapi arazzo graph --workflow adoptPet --direction LR workflow.arazzo.yaml
  openapi arazzo graph workflow.arazzo.yaml workflow.dot && dot -Tsvg workflow.dot -o workflow.svg`,
	Args: cmdutil.StdinOrFileArgs(1, 2),
	Run:  runGraph,
}

// graphFlags are the flags of the graph command.
type graphFlags struct {
	format    string
	workflow  string
	direction string
}

// graphIO are the streams used by the graph command.
type graphIO struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

var arazzoGraphFlags graphFlags

func init() {
	graphCmd.Flags().StringVarP(&arazzoGraphFlags.format, "format", "f", "", "Output format: mermaid or dot (default mermaid, or dot for .dot and .gv output files)")
	graphCmd.Flags().StringVarP(&arazzoGraphFlags.workflow, "workflow", "w", "", "Only render the workflow with this workflowId")
	graphCmd.Flags().StringVarP(&arazzoGraphFlags.direction, "direction", "d", "TB", "Layout direction: TB (top to bottom) or LR (left to right)")
}

func runGraph(cmd *cobra.Command, args []string) {
	if err := graphArazzo(cmd.Context(), args, arazzoGraphFlags, graphIO{stdin: os.Stdin, stdout: os.Stdout, stderr: os.Stderr}); err != nil {
		cmdutil.Die(err)
	}
}

func graphArazzo(ctx context.Context, args []string, flags graphFlags, streams graphIO) error {
	inputFile := cmdutil.InputFileFromArgs(args)
	outputFile := cmdutil.ArgAt(args, 1, "")

	format := graph.Format(strings.ToLower(flags.format))
	if format == "" {
		format = graph.FormatMermaid
		switch strings.ToLower(filepath.Ext(outputFile)) {
		case ".dot", ".gv":
			format = graph.FormatDOT
		}
	}
	if format != graph.FormatMermaid && format != graph.FormatDOT {
		return fmt.Errorf("unknown format `%s`, expected mermaid or dot", flags.format)
	}

	var data []byte
	var err error
	if cmdutil.IsStdin(inputFile) {
		fmt.Fprintf(streams.stderr, "Rendering Arazzo document from stdin\n")
		data, err = io.ReadAll(streams.stdin)
		if err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}
	} else {
		cleanInputFile := filepath.Clean(inputFile)
		fmt.Fprintf(streams.stderr, "Rendering Arazzo document: %s\n", cleanInputFile)
		data, err = os.ReadFile(cleanInputFile)
		if err != nil {
			return fmt.Errorf("failed to open input file: %w", err)
		}
	}

	doc, _, err := arazzo.Unmarshal(ctx, bytes.NewReader(data), arazzo.WithSkipValidation())
	if err != nil {
		return fmt.Errorf("failed to unmarshal Arazzo document: %w", err)
	}
	if doc == nil {
		return errors.New("failed to parse Arazzo document: document is nil")
	}

	g, err := graph.Build(doc, graph.Options{WorkflowID: flags.workflow, Direction: graph.Direction(strings.ToUpper(flags.direction))})
	if err != nil {
		return err
	}

	var out bytes.Buffer
	if err := g.Write(&out, format); err != nil {
		return err
	}

	if outputFile == "" {
		_, err := streams.stdout.Write(out.Bytes())
		return err
	}

	cleanOutputFile := filepath.Clean(outputFile)
	if err := os.WriteFile(cleanOutputFile, out.Bytes(), 0o644); err != nil { //nolint:gosec // output files are user specified
		return fmt.Errorf("failed to write output file: %w", err)
	}
	fmt.Fprintf(streams.stderr, "📄 Diagram written to: %s\n", cleanOutputFile)
	return nil
}

// GetGraphCommand returns the graph command for external use
func GetGraphCommand() *cobra.Command {
	return graphCmd
}
