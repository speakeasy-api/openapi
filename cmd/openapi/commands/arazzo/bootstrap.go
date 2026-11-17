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
	"github.com/speakeasy-api/openapi/cmd/openapi/commands/cmdutil"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/spf13/cobra"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap [output-file]",
	Short: "Create a new Arazzo document, either from an OpenAPI document or as a best practice example",
	Long: `Create a new Arazzo document.

With --from, generates starter workflows for the operations in an OpenAPI document:
- One workflow per tag, calling the tag's operations in document order. Operations without tags
  go in a workflow named ` + "`" + arazzo.UntaggedWorkflowID + "`" + `, and operations with several tags are in each tag's workflow.
- Each step targets its operation by operationId (or operationPath when it doesn't have one), passes the
  required parameters and request body, and succeeds on the operation's documented 2xx status code.
- Values come from workflow inputs typed from the operation's schemas. When an earlier step's successful
  response declares an OpenAPI link to the operation, the value comes from that step's outputs instead.

Use --tag to only generate the workflows for some tags, or --operations to generate a single workflow
calling the listed operations in order.

The OpenAPI document is referenced by a sourceDescription whose url is relative to the output file
(or as given when writing to stdout), so the generated document can be checked with
` + "`openapi arazzo validate`" + ` straight away.

Without --from, creates an example Arazzo document demonstrating best practices: workflow inputs and
outputs, steps that pass values to each other, success criteria, and reusable components.

Examples:
  # Create an example Arazzo document
  openapi arazzo bootstrap

  # Generate workflows for each tag of an OpenAPI document
  openapi arazzo bootstrap --from openapi.yaml workflows.arazzo.yaml

  # Only generate the workflow for the pets tag
  openapi arazzo bootstrap --from openapi.yaml --tag pets workflows.arazzo.yaml

  # Generate a single workflow calling three operations in order
  openapi arazzo bootstrap --from openapi.yaml --operations createPet,getPet,adoptPet --workflow-id adoptPet workflows.arazzo.yaml`,
	Args: cobra.MaximumNArgs(1),
	Run:  runBootstrap,
}

// bootstrapFlags are the flags of the bootstrap command.
type bootstrapFlags struct {
	from       string
	tags       []string
	operations []string
	workflowID string
	sourceName string
}

var arazzoBootstrapFlags bootstrapFlags

func init() {
	bootstrapCmd.Flags().StringVar(&arazzoBootstrapFlags.from, "from", "", "OpenAPI document to generate workflows from")
	bootstrapCmd.Flags().StringSliceVar(&arazzoBootstrapFlags.tags, "tag", nil, "only generate the workflows for these tags (repeatable or comma separated, `"+arazzo.UntaggedWorkflowID+"` selects untagged operations)")
	bootstrapCmd.Flags().StringSliceVar(&arazzoBootstrapFlags.operations, "operations", nil, "generate a single workflow calling these operationIds in order (comma separated)")
	bootstrapCmd.Flags().StringVar(&arazzoBootstrapFlags.workflowID, "workflow-id", "", "workflowId of the workflow generated for --operations (default `"+arazzo.DefaultBootstrapWorkflowID+"`)")
	bootstrapCmd.Flags().StringVar(&arazzoBootstrapFlags.sourceName, "source-name", "", "name of the OpenAPI sourceDescription (default derived from the OpenAPI document's title)")
}

func runBootstrap(cmd *cobra.Command, args []string) {
	if err := bootstrapArazzo(cmd.Context(), args, arazzoBootstrapFlags, os.Stdout, os.Stderr); err != nil {
		cmdutil.Die(err)
	}
}

func bootstrapArazzo(ctx context.Context, args []string, flags bootstrapFlags, stdout, stderr io.Writer) error {
	outputFile := cmdutil.ArgAt(args, 0, "")

	var doc *arazzo.Arazzo
	if flags.from == "" {
		if len(flags.tags) > 0 || len(flags.operations) > 0 || flags.workflowID != "" || flags.sourceName != "" {
			return errors.New("--tag, --operations, --workflow-id and --source-name require --from")
		}
		doc = arazzo.Bootstrap()
	} else {
		var err error
		doc, err = bootstrapFromOpenAPI(ctx, outputFile, flags, stderr)
		if err != nil {
			return err
		}
	}

	var out bytes.Buffer
	if err := arazzo.Marshal(ctx, doc, &out); err != nil {
		return fmt.Errorf("failed to write Arazzo document: %w", err)
	}

	if outputFile == "" {
		_, err := stdout.Write(out.Bytes())
		return err
	}

	cleanOutputFile := filepath.Clean(outputFile)
	if err := os.WriteFile(cleanOutputFile, out.Bytes(), 0o644); err != nil { //nolint:gosec // output files are user specified
		return fmt.Errorf("failed to write output file: %w", err)
	}
	fmt.Fprintf(stderr, "📄 Arazzo document written to: %s\n", cleanOutputFile)
	return nil
}

func bootstrapFromOpenAPI(ctx context.Context, outputFile string, flags bootstrapFlags, stderr io.Writer) (*arazzo.Arazzo, error) {
	from := filepath.Clean(flags.from)
	fmt.Fprintf(stderr, "Generating Arazzo workflows from: %s\n", from)

	data, err := os.ReadFile(from)
	if err != nil {
		return nil, fmt.Errorf("failed to open OpenAPI document: %w", err)
	}

	source, _, err := openapi.Unmarshal(ctx, bytes.NewReader(data), openapi.WithSkipValidation())
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal OpenAPI document: %w", err)
	}
	if source == nil {
		return nil, errors.New("failed to parse OpenAPI document: document is nil")
	}

	absFrom, err := filepath.Abs(from)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve OpenAPI document path: %w", err)
	}
	if _, err := source.ResolveAllReferences(ctx, openapi.ResolveAllOptions{OpenAPILocation: absFrom}); err != nil {
		fmt.Fprintf(stderr, "⚠️  Some references could not be resolved, so parameters, request bodies or links using them are skipped: %v\n", err)
	}

	sourceURL, err := relativeSourceURL(outputFile, absFrom)
	if err != nil {
		return nil, err
	}

	return arazzo.BootstrapFromOpenAPI(source, arazzo.BootstrapOptions{
		SourceURL:    sourceURL,
		SourceName:   flags.sourceName,
		Tags:         flags.tags,
		OperationIDs: flags.operations,
		WorkflowID:   flags.workflowID,
	})
}

// relativeSourceURL returns the location of the OpenAPI document relative to the output file, or to the current
// directory when writing to stdout.
func relativeSourceURL(outputFile, absFrom string) (string, error) {
	base := "."
	if outputFile != "" {
		base = filepath.Dir(outputFile)
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("failed to resolve output directory: %w", err)
	}

	rel, err := filepath.Rel(absBase, absFrom)
	if err != nil {
		// Different volumes on Windows, so use the absolute path.
		return filepath.ToSlash(absFrom), nil //nolint:nilerr
	}

	rel = filepath.ToSlash(rel)
	if !strings.HasPrefix(rel, "../") {
		rel = "./" + rel
	}
	return rel, nil
}

// GetBootstrapCommand returns the bootstrap command for external use
func GetBootstrapCommand() *cobra.Command {
	return bootstrapCmd
}
