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

Tip: use where(isComponent) or where(isInline) to filter the schemas source:
  schemas | where(isComponent)
  schemas | where(isInline)

TRAVERSAL STAGES
----------------
Graph navigation stages replace the current result set by following edges
in the schema reference graph.

  refs-out            Outgoing references, 1 hop (with edge annotations)
  refs-out(*)         Full transitive closure of outgoing references
  refs-in             Incoming references, 1 hop (with edge annotations)
  refs-in(*)          Full transitive closure of incoming references
  properties          Expand to property sub-schemas (flattens allOf; with edge annotations)
  properties(*)       Recursive properties (follows $refs, flattens allOf,
                      expands oneOf/anyOf with qualified from paths)
  members             Expand allOf/oneOf/anyOf children, or group rows into schemas
  items               Expand to array items schema (checks allOf; with edge annotations)
  parent              Navigate to structural parent schema (via graph in-edges)
  to-operations       Schemas → operations that use them
  to-schemas          Operations → schemas they touch
  path(A, B)          Shortest path between two named schemas (auto-tries both directions)
  connected           Full connected component (schemas + operations)
  blast-radius        Ancestors + all affected operations (change impact)
  neighbors           Bidirectional neighborhood, 1 hop (default)
  neighbors(N)        Bidirectional neighborhood within N hops
  neighbors(*)        Full bidirectional closure

NAVIGATION STAGES
-----------------
Navigate between operations and their sub-components. These stages produce
typed rows that can be filtered, projected, and navigated back to the source.

  parameters            Operation parameters (yields ParameterRow)
  responses             Operation responses (yields ResponseRow)
  request-body          Operation request body (yields RequestBodyRow)
  content-types         Content types from responses or request body (yields ContentTypeRow)
  headers               Response headers (yields HeaderRow)
  to-schema             Extract schema from parameter, content-type, or header (bridges to graph)
  operation             Back-navigate to source operation
  security              Operation security requirements (inherits global when not overridden)

ANALYSIS STAGES
---------------

  orphans            Schemas with no incoming refs and no operation usage
  leaves             Schemas with no outgoing refs (leaf/terminal nodes)
  cycles             Strongly connected components (actual reference cycles)
  clusters           Weakly connected component grouping
  cross-tag          Component schemas used by operations across multiple tags
  shared-refs        Schemas shared by ALL operations in result set
  shared-refs(N)     Schemas shared by at least N operations

FILTER & TRANSFORM STAGES
--------------------------

  where(expr)            Filter rows by predicate expression
  select <fields>        Project specific fields (comma-separated)
  sort-by(field)         Sort ascending by field
  sort-by(field, desc)   Sort descending by field
  take(N)                Limit to first N results
  last(N)                Limit to last N results
  sample(N)              Deterministic pseudo-random sample of N rows
  highest(N, field)      Sort descending by field and take N (shorthand)
  lowest(N, field)       Sort ascending by field and take N (shorthand)
  unique                 Deduplicate rows (by projected fields when select is active)
  group-by(field)        Group rows and aggregate counts
  group-by(field, name_field)  Group with custom name field for aggregation
  length                 Count rows (terminal — returns a single number)
  let $var = expr        Bind expression result to a variable for later stages

FUNCTION DEFINITIONS & MODULES
-------------------------------
Define reusable pipeline fragments:

  def hot: where(inDegree > 10);
  def impact($name): where(name == $name) | blast-radius;

  Syntax: def name: body;
          def name($p1, $p2): body;

Load definitions from .oq files:

  include "stdlib.oq";

  Search paths: current directory, then ~/.config/oq/

META STAGES
-----------

  explain              Print the query execution plan instead of running it
  fields               List available fields for the current result kind
  to-yaml              Output raw YAML nodes from underlying spec objects (pipe into yq)
  format(fmt)          Set output format: table, json, markdown, or toon

SCHEMA FIELDS
-------------

Graph-level (pre-computed):

  Field             Type     Description
  ─────             ────     ───────────
  name              string   Component name or JSON pointer
  type              string   Schema type (object, array, string, ...)
  depth             int      Max nesting depth
  inDegree         int      Number of schemas referencing this one
  outDegree        int      Number of schemas this references
  unionWidth       int      oneOf + anyOf + allOf member count
  allOfCount       int      Number of allOf members
  oneOfCount       int      Number of oneOf members
  anyOfCount       int      Number of anyOf members
  propertyCount    int      Number of properties
  properties        array    Property names (for 'contains' filtering)
  isComponent      bool     In #/components/schemas
  isInline         bool     Defined inline
  isCircular       bool     Part of a circular reference chain
  hasRef           bool     Has a $ref
  hash              string   Content hash
  path              string   JSON pointer in document
  opCount          int      Number of operations using this schema
  tagCount         int      Number of distinct tags across operations

Content-level (from schema object):

  Field                        Type     Description
  ─────                        ────     ───────────
  description                  string   Schema description text
  hasDescription              bool     Whether description is non-empty
  title                        string   Schema title
  hasTitle                    bool     Whether title is non-empty
  format                       string   Format hint (date-time, uuid, int32, ...)
  pattern                      string   Regex validation pattern
  nullable                     bool     Nullable flag
  readOnly                    bool     Read-only flag
  writeOnly                   bool     Write-only flag
  deprecated                   bool     Deprecated flag
  uniqueItems                 bool     Array unique items constraint
  hasDiscriminator            bool     Has discriminator object
  discriminatorProperty       string   Discriminator property name
  discriminatorMappingCount  int      Number of discriminator mappings
  requiredCount               int      Number of required properties
  enumCount                   int      Number of enum values
  hasDefault                  bool     Has a default value
  hasExample                  bool     Has example(s)
  minimum                      int?     Minimum numeric value (null if unset)
  maximum                      int?     Maximum numeric value (null if unset)
  minLength                   int?     Minimum string length (null if unset)
  maxLength                   int?     Maximum string length (null if unset)
  minItems                    int?     Minimum array items (null if unset)
  maxItems                    int?     Maximum array items (null if unset)
  minProperties               int?     Minimum object properties (null if unset)
  maxProperties               int?     Maximum object properties (null if unset)
  extensionCount              int      Number of x- extensions
  contentEncoding             string   Content encoding (base64, ...)
  content_mediaType           string   Content media type

OPERATION FIELDS
----------------

  Field               Type     Description
  ─────               ────     ───────────
  name                string   operationId or "METHOD /path"
  method              string   HTTP method (GET, POST, ...)
  path                string   URL path
  operationId        string   operationId
  schemaCount        int      Total reachable schema count
  componentCount     int      Reachable component schema count
  tag                 string   First tag
  tags                string   All tags (comma-separated)
  parameterCount     int      Number of parameters
  deprecated          bool     Whether the operation is deprecated
  description         string   Operation description
  summary             string   Operation summary
  responseCount      int      Number of response status codes
  hasErrorResponse  bool     Has 4xx/5xx or default response
  hasRequestBody    bool     Has a request body
  securityCount      int      Number of security requirements

EDGE ANNOTATION FIELDS
----------------------
Available on rows produced by traversal stages (refs-out, refs-in,
properties, members, items). Use 'parent' to navigate back to the
source schema.

  Field             Type     Description
  ─────             ────     ───────────
  via               string   Structural edge kind: property, items, allOf, oneOf, ...
  key               string   Structural edge label: property name, array index, etc.
  from              string   Source schema name (the schema containing the relationship)
  target            string   Seed schema name (the schema that initiated the traversal)
  bfsDepth         int      BFS depth from seed (populated by properties(*))

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
  hasSchema          bool     Whether the parameter has a schema
  allowEmptyValue   bool     Whether empty values are allowed
  allowReserved      bool     Whether reserved characters are allowed
  operation           string   Source operation (operationId or METHOD /path)

RESPONSE FIELDS
---------------

  Field               Type     Description
  ─────               ────     ───────────
  statusCode         string   HTTP status code (200, 404, default, ...)
  name                string   Response name
  description         string   Response description
  contentTypeCount  int      Number of content types
  headerCount        int      Number of response headers
  linkCount          int      Number of links
  hasContent         bool     Whether response has content
  operation           string   Source operation

REQUEST BODY FIELDS
-------------------

  Field               Type     Description
  ─────               ────     ───────────
  name                string   Request body name
  description         string   Request body description
  required            bool     Whether the request body is required
  contentTypeCount  int      Number of content types
  operation           string   Source operation

CONTENT-TYPE FIELDS
-------------------

  Field               Type     Description
  ─────               ────     ───────────
  mediaType          string   Media type (application/json, text/event-stream, ...)
  name                string   Content type name
  hasSchema          bool     Whether it has a schema
  hasEncoding        bool     Whether it has encoding info
  hasExample         bool     Whether it has an example
  statusCode         string   Source response status code (if from response)
  operation           string   Source operation

HEADER FIELDS
-------------

  Field               Type     Description
  ─────               ────     ───────────
  name                string   Header name
  description         string   Header description
  required            bool     Whether the header is required
  deprecated          bool     Whether the header is deprecated
  hasSchema          bool     Whether the header has a schema
  statusCode         string   Source response status code
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
  bearerFormat       string   Bearer token format hint, e.g. JWT (http only)
  description         string   Scheme description
  hasFlows           bool     Whether OAuth2 flows are defined (oauth2 only)
  deprecated          bool     Whether the scheme is deprecated

SECURITY REQUIREMENT FIELDS
----------------------------
Available from operations | security stage. Inherits global security when
the operation has no per-operation override. An explicit empty security: []
on an operation means "no security" (yields zero rows).

  Field               Type     Description
  ─────               ────     ───────────
  schemeName         string   Security scheme name
  schemeType         string   Resolved scheme type (apiKey, http, oauth2, ...)
  scopes              array    Required OAuth2 scopes
  scopeCount         int      Number of required scopes
  operation           string   Source operation

EXPRESSIONS
-----------
The expression language is used in where(), let, and if-then-else:

  Comparison:     ==  !=  >  <  >=  <=
  Logical:        and  or  not
  Alternative:    //  (alias: default) — returns left if truthy, else right
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

Use the to-yaml stage to extract raw YAML nodes from the underlying spec objects.
Schema rows use full JSON pointer paths as keys. Navigation rows use contextual
compound keys (e.g., "listUsers/200" for responses, "createUser/parameters/limit"
for parameters). Pipe into yq for content-level queries:
  openapi spec query 'schemas | where(name == "Pet") | to-yaml' spec.yaml | yq '.properties | keys'
  openapi spec query 'operations | take(1) | responses | to-yaml' spec.yaml

EXAMPLES
--------

Schema analysis:

  # Deeply nested component schemas
  components.schemas | sort-by(depth, desc) | take(10) | select name, depth

  # Most referenced schemas
  components.schemas | sort-by(inDegree, desc) | take(10) | select name, inDegree

  # Dead components — defined but never referenced
  components.schemas | orphans | select name

  # Circular references
  schemas | where(isCircular) | select name, path

  # Blast radius — what breaks if I change this schema?
  schemas | where(name == "Error") | blast-radius | length

  # Depth-limited traversal — see 2 hops from a schema
  schemas | where(name == "User") | refs-out(2) | select name, type

  # Edge annotations — how a schema references others
  schemas | where(name == "Pet") | refs-out | select name, via, key, from

  # Schemas containing a property named "email"
  schemas | where(properties contains "email") | select name

  # Schemas with properties matching a pattern (via traversal)
  schemas | properties | where(key matches "(?i)date") | parent | unique | select name

  # Schemas with names starting with "Error"
  components.schemas | where(name startswith "Error") | select name, type

Operations & navigation:

  # Operation sprawl — most complex endpoints
  operations | sort-by(schemaCount, desc) | take(10) | select name, schemaCount

  # Find SSE/streaming endpoints
  operations | responses | content-types | where(mediaType == "text/event-stream") | operation | unique

  # All content types across the API
  operations | responses | content-types | select mediaType | unique | sort-by(mediaType)

  # Deprecated parameters
  operations | parameters | where(deprecated) | select name, in, operation

  # Operations accepting multipart uploads
  operations | request-body | content-types | where(mediaType matches "multipart/") | operation | unique

  # Response headers
  operations | responses | headers | select name, required, statusCode, operation

  # Drill into a response schema
  operations | where(name == "createUser") | request-body | content-types | to-schema | refs-out(2) | to-yaml

  # Group responses by status code (showing operation names)
  operations | responses | group-by(statusCode, operation)

Security:

  # List all security schemes
  components.security-schemes | select name, type, scheme

  # Operations using OAuth2
  operations | security | where(schemeType == "oauth2") | select schemeName, scopes, operation

  # Operations with no security
  operations | security | length  # compare with: operations | length

Content auditing:

  # OneOf unions missing discriminator
  components.schemas | where(unionWidth > 0 and not hasDiscriminator) | select name, unionWidth

  # Schemas missing descriptions
  components.schemas | where(not hasDescription) | select name, type

  # Operations missing error responses
  operations | where(not hasErrorResponse) | select name, method, path

  # Duplicate inline schemas (same hash)
  schemas | where(isInline) | group-by(hash) | where(count > 1)

Advanced:

  # Variable binding
  schemas | where(name == "Pet") | let $pet = name | refs-out | where(name != $pet) | select name

  # User-defined functions
  def hot: where(inDegree > 10);
  schemas | where(isComponent) | hot | select name, inDegree

  # Raw YAML extraction (pipe into yq)
  operations | take(1) | responses | to-yaml
  schemas | where(name == "Pet") | to-yaml | yq '.Pet.properties | keys'
`
