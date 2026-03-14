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

  schemas                      All schemas (component + inline)
  operations                   All operations
  components.schemas           Component schemas only
  components.parameters        Reusable parameter definitions
  components.responses         Reusable response definitions
  components.request-bodies    Reusable request body definitions
  components.headers           Reusable header definitions
  components.security-schemes  Security scheme definitions

Tip: use select(is_component) or select(is_inline) to filter the schemas source:
  schemas | select(is_component)
  schemas | select(is_inline)

TRAVERSAL STAGES
----------------
Graph navigation stages replace the current result set by following edges
in the schema reference graph.

  references          Direct outgoing references (1 hop, with edge annotations)
  referenced-by           Direct incoming references (1 hop, with edge annotations)
  descendants         Transitive closure of outgoing references (all hops)
  descendants(N)      Depth-limited descendants: only follow N hops
  ancestors           Transitive closure of incoming references
  ancestors(N)        Depth-limited ancestors: only follow N hops
  properties        Expand to property sub-schemas (with edge annotations)
  union-members     Expand allOf/oneOf/anyOf children (with edge annotations)
  items             Expand to array items schema (with edge annotations)
  parent            Navigate to structural parent schema (via graph in-edges)
  ops               Schemas → operations that use them
  schemas           Operations → schemas they touch
  path(A; B)        Shortest path between two named schemas
  connected         Full connected component (schemas + operations)
  blast-radius      Ancestors + all affected operations (change impact)
  neighbors(N)      Bidirectional neighborhood within N hops

NAVIGATION STAGES
-----------------
Navigate between operations and their sub-components. These stages produce
typed rows that can be filtered, projected, and navigated back to the source.

  parameters            Operation parameters (yields ParameterRow)
  responses             Operation responses (yields ResponseRow)
  request-body          Operation request body (yields RequestBodyRow)
  content-types         Content types from responses or request body (yields ContentTypeRow)
  headers               Response headers (yields HeaderRow)
  schema                Extract schema from parameter, content-type, or header (bridges to graph)
  operation             Back-navigate to source operation
  security              Operation security requirements (inherits global when not overridden)

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
  unique               Deduplicate rows (by projected fields when pick is active)
  group_by(field)      Group rows and aggregate counts
  group_by(field; name_field)  Group with custom name field for aggregation
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
  properties        array    Property names (for 'contains' filtering)
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
Available on rows produced by 1-hop traversal stages (references, referenced-by,
properties, union-members, items). Use 'parent' to navigate back to the
source schema.

  Field             Type     Description
  ─────             ────     ───────────
  via               string   Structural edge kind: property, items, allOf, oneOf, ...
  key               string   Structural edge label: property name, array index, etc.
  from              string   Source schema name (the schema containing the relationship)
  target            string   Seed schema name (the schema that initiated the traversal)
  bfs_depth         int      BFS depth from seed (populated by descendants(N), ancestors(N))

PARAMETER FIELDS
----------------

  Field               Type     Description
  ─────               ────     ───────────
  name                string   Parameter name
  in                  string   Location: query, header, path, cookie
  required            bool     Whether the parameter is required
  deprecated          bool     Whether the parameter is deprecated
  description         string   Parameter description
  style               string   Serialization style
  explode             bool     Whether arrays/objects generate separate params
  has_schema          bool     Whether the parameter has a schema
  allow_empty_value   bool     Whether empty values are allowed
  allow_reserved      bool     Whether reserved characters are allowed
  operation           string   Source operation (operationId or METHOD /path)

RESPONSE FIELDS
---------------

  Field               Type     Description
  ─────               ────     ───────────
  status_code         string   HTTP status code (200, 404, default, ...)
  name                string   Response name
  description         string   Response description
  content_type_count  int      Number of content types
  header_count        int      Number of response headers
  link_count          int      Number of links
  has_content         bool     Whether response has content
  operation           string   Source operation

REQUEST BODY FIELDS
-------------------

  Field               Type     Description
  ─────               ────     ───────────
  name                string   Request body name
  description         string   Request body description
  required            bool     Whether the request body is required
  content_type_count  int      Number of content types
  operation           string   Source operation

CONTENT-TYPE FIELDS
-------------------

  Field               Type     Description
  ─────               ────     ───────────
  media_type          string   Media type (application/json, text/event-stream, ...)
  name                string   Content type name
  has_schema          bool     Whether it has a schema
  has_encoding        bool     Whether it has encoding info
  has_example         bool     Whether it has an example
  status_code         string   Source response status code (if from response)
  operation           string   Source operation

HEADER FIELDS
-------------

  Field               Type     Description
  ─────               ────     ───────────
  name                string   Header name
  description         string   Header description
  required            bool     Whether the header is required
  deprecated          bool     Whether the header is deprecated
  has_schema          bool     Whether the header has a schema
  status_code         string   Source response status code
  operation           string   Source operation

SECURITY SCHEME FIELDS
----------------------
Available from components.security-schemes source.

  Field               Type     Description
  ─────               ────     ───────────
  name                string   Security scheme name (component key)
  type                string   Scheme type: apiKey, http, oauth2, openIdConnect, mutualTLS
  in                  string   API key location: header, query, cookie (apiKey only)
  scheme              string   HTTP auth scheme: bearer, basic, etc. (http only)
  bearer_format       string   Bearer token format hint, e.g. JWT (http only)
  description         string   Scheme description
  has_flows           bool     Whether OAuth2 flows are defined (oauth2 only)
  deprecated          bool     Whether the scheme is deprecated

SECURITY REQUIREMENT FIELDS
----------------------------
Available from operations | security stage. Inherits global security when
the operation has no per-operation override. An explicit empty security: []
on an operation means "no security" (yields zero rows).

  Field               Type     Description
  ─────               ────     ───────────
  scheme_name         string   Security scheme name
  scheme_type         string   Resolved scheme type (apiKey, http, oauth2, ...)
  scopes              array    Required OAuth2 scopes
  scope_count         int      Number of required scopes
  operation           string   Source operation

EXPRESSIONS
-----------
The expression language is used in select(), let, and if-then-else:

  Comparison:     ==  !=  >  <  >=  <=
  Logical:        and  or  not
  Alternative:    //  (returns left if truthy, else right)
  Predicates:     has(<field>)  — true if field is non-null/non-zero
  Infix:          <expr> matches "<regex>"  — regex match
                  <expr> contains "<str>"  — substring/array membership
                  <expr> startswith "<str>"  — prefix match
                  <expr> endswith "<str>"  — suffix match
  String funcs:   lower(), upper(), trim(), len(), count()
                  replace(), split()
  Arithmetic:     +  -  *  /
  Conditional:    if <cond> then <expr> else <expr> end
                  if <cond> then <expr> elif <cond> then <expr> else <expr> end
  Interpolation:  "\(<expr>)" inside double-quoted strings
  Grouping:       ( ... )
  Literals:       "string"  'literal'  42  true  false
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
Schema rows use full JSON pointer paths as keys. Navigation rows use contextual
compound keys (e.g., "listUsers/200" for responses, "createUser/parameters/limit"
for parameters). Pipe into yq for content-level queries:
  openapi spec query 'schemas | select(name == "Pet") | emit' spec.yaml | yq '.properties | keys'
  openapi spec query 'operations | first(1) | responses | emit' spec.yaml

EXAMPLES
--------

Schema analysis:

  # Deeply nested component schemas
  components.schemas | sort_by(depth; desc) | first(10) | pick name, depth

  # Most referenced schemas
  components.schemas | sort_by(in_degree; desc) | first(10) | pick name, in_degree

  # Dead components — defined but never referenced
  components.schemas | orphans | pick name

  # Circular references
  schemas | select(is_circular) | pick name, path

  # Blast radius — what breaks if I change this schema?
  schemas | select(name == "Error") | blast-radius | length

  # Depth-limited traversal — see 2 hops from a schema
  schemas | select(name == "User") | descendants(2) | pick name, type

  # Edge annotations — how a schema references others
  schemas | select(name == "Pet") | references | pick name, via, key, from

  # Schemas containing a property named "email"
  schemas | select(properties contains "email") | pick name

  # Schemas with properties matching a pattern (via traversal)
  schemas | properties | select(key matches "(?i)date") | parent | unique | pick name

  # Schemas with names starting with "Error"
  components.schemas | select(name startswith "Error") | pick name, type

Operations & navigation:

  # Operation sprawl — most complex endpoints
  operations | sort_by(schema_count; desc) | first(10) | pick name, schema_count

  # Find SSE/streaming endpoints
  operations | responses | content-types | select(media_type == "text/event-stream") | operation | unique

  # All content types across the API
  operations | responses | content-types | pick media_type | unique | sort_by(media_type)

  # Deprecated parameters
  operations | parameters | select(deprecated) | pick name, in, operation

  # Operations accepting multipart uploads
  operations | request-body | content-types | select(media_type matches "multipart/") | operation | unique

  # Response headers
  operations | responses | headers | pick name, required, status_code, operation

  # Drill into a response schema
  operations | select(name == "createUser") | request-body | content-types | schema | descendants(2) | emit

  # Group responses by status code (showing operation names)
  operations | responses | group_by(status_code; operation)

Security:

  # List all security schemes
  components.security-schemes | pick name, type, scheme

  # Operations using OAuth2
  operations | security | select(scheme_type == "oauth2") | pick scheme_name, scopes, operation

  # Operations with no security
  operations | security | length  # compare with: operations | length

Content auditing:

  # OneOf unions missing discriminator
  components.schemas | select(union_width > 0 and not has_discriminator) | pick name, union_width

  # Schemas missing descriptions
  components.schemas | select(not has_description) | pick name, type

  # Operations missing error responses
  operations | select(not has_error_response) | pick name, method, path

  # Duplicate inline schemas (same hash)
  schemas | select(is_inline) | group_by(hash) | select(count > 1)

Advanced:

  # Variable binding
  schemas | select(name == "Pet") | let $pet = name | descendants | select(name != $pet) | pick name

  # User-defined functions
  def hot: select(in_degree > 10);
  schemas | select(is_component) | hot | pick name, in_degree

  # Raw YAML extraction (pipe into yq)
  operations | first(1) | responses | emit
  schemas | select(name == "Pet") | emit | yq '.Pet.properties | keys'
`
