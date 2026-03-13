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
  # Deeply nested components (jq-style syntax)
  openapi spec query 'schemas.components | sort_by(depth; desc) | first(10) | pick name, depth' petstore.yaml

  # Pipe from stdin
  cat spec.yaml | openapi spec query 'schemas | count'

  # Explicit stdin
  openapi spec query 'schemas | count' -

  # Filter with select()
  openapi spec query 'schemas | select(union_width > 0) | sort_by(union_width; desc) | first(10)' petstore.yaml

  # Dead components (no incoming references)
  openapi spec query 'schemas.components | select(in_degree == 0) | pick name' petstore.yaml

  # Variable binding — exclude seed from reachable results
  openapi spec query 'schemas | select(name == "Pet") | let $pet = name | reachable | select(name != $pet)' petstore.yaml

  # User-defined functions
  openapi spec query 'def hot: select(in_degree > 5); schemas.components | hot | pick name' petstore.yaml

  # Alternative operator — fallback for null/falsy values
  openapi spec query 'schemas | select(name // "none" != "none")' petstore.yaml

  # If-then-else conditional
  openapi spec query 'schemas | select(if is_component then depth > 3 else true end)' petstore.yaml

  # Blast radius
  openapi spec query 'schemas.components | select(name == "Error") | blast-radius | length' petstore.yaml

  # Explain a query plan
  openapi spec query 'schemas.components | select(depth > 5) | sort_by(depth; desc) | explain' petstore.yaml

Pipeline stages (jq-style):
  Source:     schemas, schemas.components, schemas.inline, operations
  Traversal:  refs-out, refs-in, reachable, ancestors, properties, union-members, items,
              parent, ops, schemas, path(A; B), connected, blast-radius, neighbors(N)
  Analysis:   orphans, leaves, cycles, clusters, tag-boundary, shared-refs
  Filter:     select(expr), pick <fields>, sort_by(field; desc), first(N), last(N),
              sample(N), top(N; field), bottom(N; field), unique, group_by(field), length
  Variables:  let $var = expr
  Functions:  def name: body;  def name($p): body;  include "file.oq";
  Meta:       explain, fields, format(table|json|markdown|toon|yaml)

  Legacy syntax (where, sort, take, head, select fields, group-by, count) is still supported.

Expression operators: ==, !=, >, <, >=, <=, and, or, not, //, has(), matches,
                      if-then-else-end, string interpolation \(expr)`,
	Args: queryArgs(),
	Run:  runQuery,
}

var queryOutputFormat string
var queryFromFile string

func init() {
	queryCmd.Flags().StringVar(&queryOutputFormat, "format", "table", "output format: table, json, markdown, toon, or yaml")
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
	case "yaml":
		output = oq.FormatYAML(result, g)
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
