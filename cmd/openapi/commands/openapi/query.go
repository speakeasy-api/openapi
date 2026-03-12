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
	Use:   "query <input-file> <query>",
	Short: "Query an OpenAPI specification using the oq pipeline language",
	Long: `Query an OpenAPI specification using the oq pipeline language to answer
structural and semantic questions about schemas and operations.

Examples:
  # Deeply nested components
  openapi spec query petstore.yaml 'schemas.components | sort depth desc | take 10 | select name, depth'

  # Wide union trees
  openapi spec query petstore.yaml 'schemas | where union_width > 0 | sort union_width desc | take 10'

  # Central components (highest in-degree)
  openapi spec query petstore.yaml 'schemas.components | sort in_degree desc | take 10 | select name, in_degree'

  # Dead components (no incoming references)
  openapi spec query petstore.yaml 'schemas.components | where in_degree == 0 | select name'

  # Operation sprawl
  openapi spec query petstore.yaml 'operations | sort schema_count desc | take 10 | select name, schema_count'

  # Circular references
  openapi spec query petstore.yaml 'schemas | where is_circular | select name, path'

  # Schema count
  openapi spec query petstore.yaml 'schemas | count'

Stdin is supported — either pipe data directly or use '-' explicitly:
  cat spec.yaml | openapi spec query - 'schemas | count'

  # Shortest path between schemas
  openapi spec query petstore.yaml 'schemas | path "Pet" "Address" | select name'

  # Top 5 most connected schemas
  openapi spec query petstore.yaml 'schemas.components | top 5 in_degree | select name, in_degree'

  # Explain a query plan
  openapi spec query petstore.yaml 'schemas.components | where depth > 5 | sort depth desc | explain'

  # List available fields
  openapi spec query petstore.yaml 'schemas | fields'

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
	inputFile := inputFileFromArgs(args)

	queryStr := ""
	if queryFromFile != "" {
		data, err := os.ReadFile(queryFromFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading query file: %v\n", err)
			os.Exit(1)
		}
		queryStr = string(data)
	} else if len(args) >= 2 {
		queryStr = args[1]
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
