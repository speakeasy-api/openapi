package openapi

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/speakeasy-api/openapi/graph"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/oq"
	"github.com/speakeasy-api/openapi/references"
	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query <query> [input-file]",
	Short: "Query an OpenAPI specification using the oq pipeline language",
	Long: `Query an OpenAPI specification using the oq pipeline language to answer
structural and semantic questions about schemas and operations.

The query argument comes first, followed by an optional input file. If no file
is given, reads from stdin.

Examples:
  # Deeply nested components
  openapi spec query 'schemas.components | sort depth desc | take 10 | select name, depth' petstore.yaml

  # Pipe from stdin
  cat spec.yaml | openapi spec query 'schemas | count'

  # Explicit stdin
  openapi spec query 'schemas | count' -

  # Wide union trees
  openapi spec query 'schemas | where union_width > 0 | sort union_width desc | take 10' petstore.yaml

  # Dead components (no incoming references)
  openapi spec query 'schemas.components | where in_degree == 0 | select name' petstore.yaml

  # Operation sprawl
  openapi spec query 'operations | sort schema_count desc | take 10 | select name, schema_count' petstore.yaml

  # Circular references
  openapi spec query 'schemas | where is_circular | select name, path' petstore.yaml

  # Shortest path between schemas
  openapi spec query 'schemas | path "Pet" "Address" | select name' petstore.yaml

  # Edge annotations
  openapi spec query 'schemas.components | where name == "Pet" | refs-out | select name, edge_kind, edge_label' petstore.yaml

  # Blast radius
  openapi spec query 'schemas.components | where name == "Error" | blast-radius | count' petstore.yaml

  # Explain a query plan
  openapi spec query 'schemas.components | where depth > 5 | sort depth desc | explain' petstore.yaml

Pipeline stages:
  Source:     schemas, schemas.components, schemas.inline, operations
  Traversal:  refs-out, refs-in, reachable, ancestors, properties, union-members, items,
              ops, schemas, path <from> <to>, connected, blast-radius, neighbors <n>
  Analysis:   orphans, leaves, cycles, clusters, tag-boundary, shared-refs
  Filter:     where <expr>, select <fields>, sort <field> [asc|desc], take/head <n>,
              sample <n>, top <n> <field>, bottom <n> <field>, unique, group-by <field>, count
  Meta:       explain, fields, format <table|json|markdown|toon>

Where expressions support: ==, !=, >, <, >=, <=, and, or, not, has(), matches`,
	Args: stdinOrFileArgs(1, 2),
	Run:  runQuery,
}

var queryOutputFormat string
var queryFromFile string

func init() {
	queryCmd.Flags().StringVar(&queryOutputFormat, "format", "table", "output format: table, json, markdown, or toon")
	queryCmd.Flags().StringVarP(&queryFromFile, "file", "f", "", "read query from file instead of argument")
}

func runQuery(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	// args[0] = query (or input file if using -f), args[1] = input file (optional)
	queryStr := ""
	inputFile := "-" // default to stdin

	if queryFromFile != "" {
		data, err := os.ReadFile(queryFromFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading query file: %v\n", err)
			os.Exit(1)
		}
		queryStr = string(data)
		// When using -f, all positional args are input files
		if len(args) > 0 {
			inputFile = args[0]
		}
	} else if len(args) >= 1 {
		queryStr = args[0]
		if len(args) >= 2 {
			inputFile = args[1]
		}
	}

	if queryStr == "" {
		fmt.Fprintf(os.Stderr, "Error: no query provided\n")
		os.Exit(1)
	}

	processor, err := NewOpenAPIProcessor(inputFile, "", false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := queryOpenAPI(ctx, processor, queryStr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func queryOpenAPI(ctx context.Context, processor *OpenAPIProcessor, queryStr string) error {
	doc, _, err := processor.LoadDocument(ctx)
	if err != nil {
		return err
	}
	if doc == nil {
		return errors.New("failed to parse OpenAPI document: document is nil")
	}

	// Build index
	idx := buildIndex(ctx, doc)

	// Build graph
	g := graph.Build(ctx, idx)

	// Execute query
	result, err := oq.Execute(queryStr, g)
	if err != nil {
		return fmt.Errorf("query error: %w", err)
	}

	// Format and output — inline format stage overrides CLI flag
	format := queryOutputFormat
	if result.FormatHint != "" {
		format = result.FormatHint
	}

	var output string
	switch format {
	case "json":
		output = oq.FormatJSON(result, g)
	case "markdown":
		output = oq.FormatMarkdown(result, g)
	case "toon":
		output = oq.FormatToon(result, g)
	default:
		output = oq.FormatTable(result, g)
	}

	fmt.Fprint(processor.stdout(), output)
	if result.IsCount {
		fmt.Fprintln(processor.stdout())
	}

	return nil
}

func buildIndex(ctx context.Context, doc *openapi.OpenAPI) *openapi.Index {
	resolveOpts := references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: ".",
	}
	return openapi.BuildIndex(ctx, doc, resolveOpts)
}
