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
structural and semantic questions about schemas, operations, parameters,
responses, content types, and headers.`,
	Example: `Queries are pipelines: source | stage | stage | ...

Pipeline stages:
  Source:      schemas, operations, components.schemas, components.parameters,
               components.responses, components.request-bodies, components.headers,
               components.security-schemes
  Navigation:  parameters, responses, request-body, content-types, headers,
               schema, operation, security
  Traversal:   refs-out, refs-in, reachable, reachable(N), ancestors, properties,
               union-members, items, parent, ops, schemas, path(A; B), connected,
               blast-radius, neighbors(N)
  Analysis:    orphans, leaves, cycles, clusters, tag-boundary, shared-refs
  Filter:      select(expr), pick <fields>, sort_by(field; desc), first(N), last(N),
               sample(N), top(N; field), bottom(N; field), unique,
               group_by(field), group_by(field; name_field), length
  Variables:   let $var = expr
  Functions:   def name: body;  def name($p): body;  include "file.oq";
  Output:      emit, format(table|json|markdown|toon)
  Meta:        explain, fields

Operators: ==, !=, >, <, >=, <=, and, or, not, //, has(), matches, contains,
           if-then-else-end, \(interpolation), lower(), upper(), len(), split()

  openapi spec query 'operations | responses | content-types | select(media_type == "text/event-stream") | operation | unique' spec.yaml
  openapi spec query 'operations | security | group_by(scheme_type; operation)' spec.yaml
  openapi spec query 'schemas | select(is_component) | sort_by(depth; desc) | first(10) | pick name, depth' spec.yaml
  openapi spec query 'operations | select(name == "createUser") | request-body | content-types | schema | reachable(2) | emit' spec.yaml
  openapi spec query 'components.security-schemes | pick name, type, scheme' spec.yaml
  cat spec.yaml | openapi spec query 'schemas | length'

For the full query language reference, run: openapi spec query-reference`,
	Args: queryArgs(),
	Run:  runQuery,
}

var queryOutputFormat string
var queryFromFile string

func init() {
	queryCmd.Flags().StringVar(&queryOutputFormat, "format", "table", "output format: table, json, markdown, or toon")
	queryCmd.Flags().StringVarP(&queryFromFile, "file", "f", "", "read query from file instead of argument")

	// Custom help template: Usage + Flags together, then Examples last
	queryCmd.SetUsageTemplate(`Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasExample}}

{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`)
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

	// Emit stage outputs raw YAML nodes, bypassing format selection
	if result.EmitYAML {
		output := oq.FormatYAML(result, g)
		fmt.Fprint(processor.stdout(), output)
		return nil
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
