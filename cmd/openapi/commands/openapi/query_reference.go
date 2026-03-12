package openapi

import (
	"fmt"

	"github.com/spf13/cobra"
)

var queryReferenceCmd = &cobra.Command{
	Use:   "query-reference",
	Short: "Print the oq query language reference",
	Long:  "Print the complete reference for the oq pipeline query language, including all stages, fields, operators, and examples.",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Print(queryReference)
	},
}

const queryReference = `oq — OpenAPI Query Language Reference
=====================================

oq is a pipeline query language for exploring OpenAPI schema graphs.
Queries are composed as left-to-right pipelines:

  source | stage | stage | ... | terminal

SOURCES
-------
The first element of every pipeline is a source that selects the initial
result set.

  schemas              All schemas (component + inline)
  schemas.components   Only component schemas (in #/components/schemas)
  schemas.inline       Only inline schemas
  operations           All operations

TRAVERSAL STAGES
----------------
Graph navigation stages replace the current result set by following edges
in the schema reference graph.

  refs-out        Direct outgoing references (1 hop)
  refs-in         Direct incoming references (1 hop)
  reachable       Transitive closure of outgoing references
  ancestors       Transitive closure of incoming references
  properties      Expand to property sub-schemas
  union-members   Expand allOf/oneOf/anyOf children
  items           Expand to array items schema
  ops             Schemas → operations that use them
  schemas         Operations → schemas they touch
  path <a> <b>   Shortest path between two named schemas

FILTER & TRANSFORM STAGES
--------------------------

  where <expr>         Filter rows by predicate expression
  select <fields>      Project specific fields (comma-separated)
  sort <field> [desc]  Sort by field (default ascending, add "desc" for descending)
  take <n>             Limit to first N results
  head <n>             Alias for take
  sample <n>           Deterministic pseudo-random sample of N rows
  top <n> <field>      Sort descending by field and take N (shorthand)
  bottom <n> <field>   Sort ascending by field and take N (shorthand)
  unique               Deduplicate rows by identity
  group-by <field>     Group rows and aggregate counts
  count                Count rows (terminal — returns a single number)

META STAGES
-----------

  explain              Print the query execution plan instead of running it
  fields               List available fields for the current result kind
  format <fmt>         Set output format: table, json, markdown, or toon

SCHEMA FIELDS
-------------

  Field             Type     Description
  ─────             ────     ───────────
  name              string   Component name or JSON pointer
  type              string   Schema type (object, array, string, ...)
  depth             int      Max nesting depth
  in_degree         int      Number of schemas referencing this one
  out_degree        int      Number of schemas this references
  union_width       int      oneOf + anyOf + allOf member count
  property_count    int      Number of properties
  is_component      bool     In #/components/schemas
  is_inline         bool     Defined inline
  is_circular       bool     Part of a circular reference chain
  has_ref           bool     Has a $ref
  hash              string   Content hash
  path              string   JSON pointer in document

OPERATION FIELDS
----------------

  Field             Type     Description
  ─────             ────     ───────────
  name              string   operationId or "METHOD /path"
  method            string   HTTP method (GET, POST, ...)
  path              string   URL path
  operation_id      string   operationId
  schema_count      int      Total reachable schema count
  component_count   int      Reachable component schema count
  tag               string   First tag
  parameter_count   int      Number of parameters
  deprecated        bool     Whether the operation is deprecated
  description       string   Operation description
  summary           string   Operation summary

WHERE EXPRESSIONS
-----------------
The where clause supports a predicate expression language:

  Comparison:   ==  !=  >  <  >=  <=
  Logical:      and  or  not
  Functions:    has(<field>)  — true if field is non-null/non-zero
                matches(<field>, "<regex>")  — regex match
  Infix:        <field> matches "<regex>"
  Grouping:     ( ... )
  Literals:     "string"  42  true  false

OUTPUT FORMATS
--------------

  table      Aligned columns with header (default)
  json       JSON array of objects
  markdown   Markdown table
  toon       TOON (Token-Oriented Object Notation) tabular format

Set via --format flag or inline format stage:
  schemas | count | format json

EXAMPLES
--------

  # Deeply nested components
  schemas.components | sort depth desc | take 10 | select name, depth

  # Wide union trees
  schemas | where union_width > 0 | sort union_width desc | take 10

  # Most referenced schemas
  schemas.components | sort in_degree desc | take 10 | select name, in_degree

  # Dead components (no incoming references)
  schemas.components | where in_degree == 0 | select name

  # Operation sprawl
  operations | sort schema_count desc | take 10 | select name, schema_count

  # Circular references
  schemas | where is_circular | select name, path

  # Schema count
  schemas | count

  # Shortest path between schemas
  schemas | path "Pet" "Address" | select name

  # Top 5 by in-degree
  schemas.components | top 5 in_degree | select name, in_degree

  # Walk an operation to find all connected schemas
  operations | where name == "GET /users" | schemas | select name, type

  # Schemas used by an operation, then find connected operations
  operations | where name == "GET /users" | schemas | ops | select name, method, path

  # Explain a query plan
  schemas.components | where depth > 5 | sort depth desc | explain

  # List available fields
  schemas | fields

  # Regex filter
  schemas | where name matches "Error.*" | select name, path

  # Complex filter
  schemas | where property_count > 3 and not is_component | select name, property_count, path
`
