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

  refs-out          Direct outgoing references (1 hop, with edge annotations)
  refs-in           Direct incoming references (1 hop, with edge annotations)
  reachable         Transitive closure of outgoing references
  ancestors         Transitive closure of incoming references
  properties        Expand to property sub-schemas (with edge annotations)
  union-members     Expand allOf/oneOf/anyOf children (with edge annotations)
  items             Expand to array items schema (with edge annotations)
  parent            Navigate back to source schema of edge annotations
  ops               Schemas → operations that use them
  schemas           Operations → schemas they touch
  path(A; B)        Shortest path between two named schemas
  connected         Full connected component (schemas + operations)
  blast-radius      Ancestors + all affected operations (change impact)
  neighbors(N)      Bidirectional neighborhood within N hops

ANALYSIS STAGES
---------------

  orphans            Schemas with no incoming refs and no operation usage
  leaves             Schemas with no outgoing refs (leaf/terminal nodes)
  cycles             Strongly connected components (actual reference cycles)
  clusters           Weakly connected component grouping
  tag-boundary       Schemas used by operations across multiple tags
  shared-refs        Schemas shared by ALL operations in result set

FILTER & TRANSFORM STAGES
--------------------------

  select(expr)         Filter rows by predicate expression (jq-style)
  pick <fields>        Project specific fields (comma-separated)
  sort_by(field)       Sort ascending by field
  sort_by(field; desc) Sort descending by field
  first(N)             Limit to first N results
  last(N)              Limit to last N results
  sample(N)            Deterministic pseudo-random sample of N rows
  top(N; field)        Sort descending by field and take N (shorthand)
  bottom(N; field)     Sort ascending by field and take N (shorthand)
  unique               Deduplicate rows by identity
  group_by(field)      Group rows and aggregate counts
  length               Count rows (terminal — returns a single number)
  let $var = expr      Bind expression result to a variable for later stages

FUNCTION DEFINITIONS & MODULES
-------------------------------
Define reusable pipeline fragments:

  def hot: select(in_degree > 10);
  def impact($name): select(name == $name) | blast-radius;

  Syntax: def name: body;
          def name($p1; $p2): body;

Load definitions from .oq files:

  include "stdlib.oq";

  Search paths: current directory, then ~/.config/oq/

META STAGES
-----------

  explain              Print the query execution plan instead of running it
  fields               List available fields for the current result kind
  emit                 Output raw YAML nodes from underlying spec objects (pipe into yq)
  format(fmt)          Set output format: table, json, markdown, or toon

SCHEMA FIELDS
-------------

Graph-level (pre-computed):

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
  op_count          int      Number of operations using this schema
  tag_count         int      Number of distinct tags across operations

Content-level (from schema object):

  Field                        Type     Description
  ─────                        ────     ───────────
  description                  string   Schema description text
  has_description              bool     Whether description is non-empty
  title                        string   Schema title
  has_title                    bool     Whether title is non-empty
  format                       string   Format hint (date-time, uuid, int32, ...)
  pattern                      string   Regex validation pattern
  nullable                     bool     Nullable flag
  read_only                    bool     Read-only flag
  write_only                   bool     Write-only flag
  deprecated                   bool     Deprecated flag
  unique_items                 bool     Array unique items constraint
  has_discriminator            bool     Has discriminator object
  discriminator_property       string   Discriminator property name
  discriminator_mapping_count  int      Number of discriminator mappings
  required_count               int      Number of required properties
  enum_count                   int      Number of enum values
  has_default                  bool     Has a default value
  has_example                  bool     Has example(s)
  minimum                      int?     Minimum numeric value (null if unset)
  maximum                      int?     Maximum numeric value (null if unset)
  min_length                   int?     Minimum string length (null if unset)
  max_length                   int?     Maximum string length (null if unset)
  min_items                    int?     Minimum array items (null if unset)
  max_items                    int?     Maximum array items (null if unset)
  min_properties               int?     Minimum object properties (null if unset)
  max_properties               int?     Maximum object properties (null if unset)
  extension_count              int      Number of x- extensions
  content_encoding             string   Content encoding (base64, ...)
  content_media_type           string   Content media type

OPERATION FIELDS
----------------

  Field               Type     Description
  ─────               ────     ───────────
  name                string   operationId or "METHOD /path"
  method              string   HTTP method (GET, POST, ...)
  path                string   URL path
  operation_id        string   operationId
  schema_count        int      Total reachable schema count
  component_count     int      Reachable component schema count
  tag                 string   First tag
  tags                string   All tags (comma-separated)
  parameter_count     int      Number of parameters
  deprecated          bool     Whether the operation is deprecated
  description         string   Operation description
  summary             string   Operation summary
  response_count      int      Number of response status codes
  has_error_response  bool     Has 4xx/5xx or default response
  has_request_body    bool     Has a request body
  security_count      int      Number of security requirements

EDGE ANNOTATION FIELDS
----------------------
Available on rows produced by 1-hop traversal stages (refs-out, refs-in,
properties, union-members, items). Use 'parent' to navigate back to the
source schema.

  Field             Type     Description
  ─────             ────     ───────────
  via               string   Edge type: property, items, allOf, oneOf, ref, ...
  key               string   Edge key: property name, array index, etc.
  from              string   Source node name

EXPRESSIONS
-----------
The expression language is used in select(), let, and if-then-else:

  Comparison:     ==  !=  >  <  >=  <=
  Logical:        and  or  not
  Alternative:    //  (returns left if truthy, else right)
  Functions:      has(<field>)  — true if field is non-null/non-zero
                  matches(<field>, "<regex>")  — regex match
  Infix:          <field> matches "<regex>"
  Conditional:    if <cond> then <expr> else <expr> end
                  if <cond> then <expr> elif <cond> then <expr> else <expr> end
  Interpolation:  "\(<expr>)" inside string literals
  Grouping:       ( ... )
  Literals:       "string"  42  true  false
  Variables:      $var (bound by let)

OUTPUT FORMATS
--------------

  table      Aligned columns with header (default)
  json       JSON array of objects
  markdown   Markdown table
  toon       TOON (Token-Oriented Object Notation) tabular format

Set via --format flag or inline format stage:
  schemas | length | format(json)

RAW YAML EXTRACTION
-------------------

Use the emit stage to extract raw YAML nodes from the underlying spec objects.
This is useful for piping into yq for content-level queries:
  openapi spec query 'schemas | select(name == "Pet") | emit' spec.yaml | yq '.properties | keys'

EXAMPLES
--------

  # Deeply nested components (jq-style)
  schemas.components | sort_by(depth; desc) | first(10) | pick name, depth

  # Wide union trees
  schemas | select(union_width > 0) | sort_by(union_width; desc) | first(10)

  # Most referenced schemas
  schemas.components | sort_by(in_degree; desc) | first(10) | pick name, in_degree

  # Dead components (no incoming references)
  schemas.components | select(in_degree == 0) | pick name

  # Operation sprawl
  operations | sort_by(schema_count; desc) | first(10) | pick name, schema_count

  # Circular references
  schemas | select(is_circular) | pick name, path

  # Schema count
  schemas | length

  # Shortest path between schemas
  schemas | path(Pet; Address) | pick name

  # Top 5 by in-degree
  schemas.components | top(5; in_degree) | pick name, in_degree

  # Walk an operation to find all connected schemas
  operations | select(name == "GET /users") | schemas | pick name, type

  # Explain a query plan
  schemas.components | select(depth > 5) | sort_by(depth; desc) | explain

  # List available fields
  schemas | fields

  # Regex filter
  schemas | select(name matches "Error.*") | pick name, path

  # Complex filter
  schemas | select(property_count > 3 and not is_component) | pick name, property_count, path

  # Edge annotations — see how Pet references other schemas
  schemas.components | select(name == "Pet") | refs-out | pick name, via, key, from

  # Parent — find schemas containing a property matching a pattern
  schemas | properties | select(key matches "(?i)date.?time") | parent | unique | emit

  # Blast radius — what breaks if I change the Error schema?
  schemas.components | select(name == "Error") | blast-radius | length

  # Neighborhood — schemas within 2 hops of Pet
  schemas.components | select(name == "Pet") | neighbors(2) | pick name

  # Orphaned schemas — unreferenced by anything
  schemas.components | orphans | pick name

  # Leaf schemas — terminal nodes with no outgoing refs
  schemas.components | leaves | pick name, in_degree

  # Detect reference cycles
  schemas | cycles

  # Discover schema clusters
  schemas.components | clusters

  # Cross-tag schemas — shared across team boundaries
  schemas | tag-boundary | pick name, tag_count

  # Schemas shared by all operations
  operations | shared-refs | pick name, op_count

  # Variable binding — find Pet's reachable schemas (excluding Pet itself)
  schemas | select(name == "Pet") | let $pet = name | reachable | select(name != $pet) | pick name

  # Alternative operator — fallback for missing values
  schemas | select(name // "unnamed" != "unnamed")

  # If-then-else — conditional filtering
  schemas | select(if is_component then depth > 3 else true end)

  # User-defined functions
  def hot: select(in_degree > 10);
  def impact($name): select(name == $name) | blast-radius;
  schemas.components | hot | pick name, in_degree

  # Load functions from a module file
  include "stdlib.oq";
  schemas.components | hot | pick name, in_degree

  # Schema content queries —

  # OneOf unions missing discriminator
  schemas.components | select(union_width > 0 and not has_discriminator) | pick name, union_width

  # Schemas missing descriptions
  schemas.components | select(not has_description) | pick name, type

  # Schemas with enums
  schemas.components | select(enum_count > 0) | pick name, enum_count

  # Operations missing error responses
  operations | select(not has_error_response) | pick name, method, path

  # Duplicate inline schemas
  schemas.inline | group_by(hash) | select(count > 1)

  # Operations with request bodies but no error handling
  operations | select(has_request_body and not has_error_response) | pick name, method, path
`
