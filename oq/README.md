# oq — OpenAPI Query Language

`oq` is a pipeline query language for exploring OpenAPI schema reference graphs. It lets you ask structural and semantic questions about schemas and operations at the command line.

## Quick Start

```bash
# Count all schemas
openapi spec query 'schemas | count' petstore.yaml

# Top 10 deepest component schemas
openapi spec query 'schemas | where(isComponent) | sort-by(depth, desc) | take(10) | select name, depth' petstore.yaml

# Dead components (unreferenced)
openapi spec query 'schemas | where(isComponent) | where(inDegree == 0) | select name' petstore.yaml
```

Stdin is supported:

```bash
cat spec.yaml | openapi spec query 'schemas | count'
```

## Pipeline Syntax

Queries are left-to-right pipelines separated by `|`:

```
source | stage | stage | ... | terminal
```

### Sources

| Source | Description |
|--------|-------------|
| `schemas` | All schemas (component + inline) |
| `operations` | All operations |

### Traversal Stages

| Stage | Description |
|-------|-------------|
| `refs-out` / `refs-out(*)` | Outgoing refs: 1-hop default, or full closure |
| `refs-in` / `refs-in(*)` | Incoming refs: 1-hop default, or full closure |
| `properties` / `properties(*)` | Property sub-schemas (allOf-flattening). `properties(*)` recursively expands through `$ref`, `oneOf`, `anyOf` with qualified `from` paths |
| `members` | allOf/oneOf/anyOf children, or expand group rows into schemas |
| `items` | Array items schema (with edge annotations) |
| `parent` | Navigate to structural parent schema (via graph in-edges) |
| `members` | Expand group rows (from `cycles`, `clusters`, `group-by`) into member schema rows |
| `to-operations` | Schemas → operations |
| `to-schemas` | Operations → schemas |
| `path(A, B)` | Shortest path between two schemas |
| `connected` | Full connected component (schemas + operations) |
| `blast-radius` | Ancestors + all affected operations |
| `neighbors(N)` | Bidirectional neighborhood within N hops |

### Navigation Stages

Navigate into the internal structure of operations. These stages produce new row types (parameters, responses, etc.) that can be filtered and inspected.

| Stage | Description |
|-------|-------------|
| `parameters` | Operation parameters |
| `responses` | Operation responses |
| `request-body` | Operation request body |
| `content-types` | Content types from response or request body |
| `headers` | Response headers |
| `to-schema` | Extract schema from parameter, content-type, or header |
| `operation` | Back-navigate to source operation |

### Analysis Stages

| Stage | Description |
|-------|-------------|
| `orphans` | Schemas with no incoming refs and no operation usage |
| `leaves` | Schemas with no outgoing refs (terminal nodes) |
| `cycles` | Strongly connected components (actual cycles) |
| `clusters` | Weakly connected component grouping |
| `cross-tag` | Schemas used by operations across multiple tags |
| `shared-refs` | Schemas shared by ALL operations in result set |

### Filter & Transform Stages

| Stage | Description |
|-------|-------------|
| `where(expr)` | Filter by predicate |
| `select f1, f2` | Project fields |
| `sort-by(field)` / `sort-by(field, desc)` | Sort (ascending by default) |
| `take(N)` | Limit to first N results |
| `last(N)` | Limit to last N results |
| `sample(N)` | Deterministic random sample |
| `highest(N, field)` | Sort desc + take |
| `lowest(N, field)` | Sort asc + take |
| `unique` | Deduplicate |
| `group-by(field)` | Group and count |
| `length` | Count rows |
| `let $var = expr` | Bind expression result to a variable |

### Meta Stages

| Stage | Description |
|-------|-------------|
| `explain` | Print query plan |
| `fields` | List available fields |
| `format(fmt)` | Set output format (table/json/markdown/toon) |
| `to-yaml` | Output raw YAML nodes from underlying spec objects |

The `to-yaml` stage uses `path` (JSON pointer) as the wrapper key for each emitted node, giving full attribution to the source location in the spec.

### Function Definitions & Modules

Define reusable functions with `def` and load them from `.oq` files with `include`:

```
# Inline definitions
def hot: where(inDegree > 10);
def impact($name): where(name == $name) | blast-radius;
schemas | where(isComponent) | hot | select name, inDegree

# Load from file
include "stdlib.oq";
schemas | where(isComponent) | hot | select name, inDegree
```

Def syntax: `def name: body;` or `def name($p1, $p2): body;`
Module search paths: current directory, then `~/.config/oq/`

## Fields

### Schema Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Component name or JSON pointer |
| `type` | string | Schema type |
| `depth` | int | Max nesting depth |
| `inDegree` | int | Incoming reference count |
| `outDegree` | int | Outgoing reference count |
| `unionWidth` | int | Union member count |
| `propertyCount` | int | Property count |
| `isComponent` | bool | In components/schemas |
| `isInline` | bool | Defined inline |
| `isCircular` | bool | Part of circular reference |
| `hasRef` | bool | Has $ref |
| `hash` | string | Content hash |
| `path` | string | JSON pointer |
| `opCount` | int | Operations using this schema |
| `tagCount` | int | Distinct tags across operations |

### Operation Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | operationId or METHOD /path |
| `method` | string | HTTP method |
| `path` | string | URL path |
| `operationId` | string | operationId |
| `schemaCount` | int | Reachable schema count |
| `componentCount` | int | Reachable component count |
| `tag` | string | First tag |
| `parameterCount` | int | Parameter count |
| `deprecated` | bool | Deprecated flag |
| `description` | string | Description |
| `summary` | string | Summary |

### Edge Annotation Fields

Available on rows produced by traversal stages (`refs-out`, `refs-in`, `properties`, `members`, `items`).

| Field | Type | Description |
|-------|------|-------------|
| `via` | string | Structural edge kind: property, items, allOf, oneOf, ... |
| `key` | string | Structural edge label: property name, array index, etc. |
| `from` | string | Source schema name (the schema containing the relationship) |
| `target` | string | Seed schema name (the schema that initiated the traversal) |
| `bfsDepth` | int | BFS depth from seed (populated by `refs-out(*)`, `refs-in(*)`) |

### Parameter Fields

Produced by the `parameters` navigation stage.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Parameter name |
| `in` | string | Location: query, header, path, cookie |
| `required` | bool | Required flag |
| `deprecated` | bool | Deprecated flag |
| `description` | string | Description |
| `style` | string | Serialization style |
| `explode` | bool | Explode flag |
| `hasSchema` | bool | Has associated schema |
| `allowEmptyValue` | bool | Allow empty value |
| `allowReserved` | bool | Allow reserved characters |
| `operation` | string | Source operation name |

### Response Fields

Produced by the `responses` navigation stage.

| Field | Type | Description |
|-------|------|-------------|
| `statusCode` | string | HTTP status code |
| `name` | string | Alias for statusCode |
| `description` | string | Response description |
| `contentTypeCount` | int | Number of content types |
| `headerCount` | int | Number of headers |
| `linkCount` | int | Number of links |
| `hasContent` | bool | Has content types |
| `operation` | string | Source operation name |

### Request Body Fields

Produced by the `request-body` navigation stage.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Always "request-body" |
| `description` | string | Request body description |
| `required` | bool | Required flag |
| `contentTypeCount` | int | Number of content types |
| `operation` | string | Source operation name |

### Content Type Fields

Produced by the `content-types` navigation stage.

| Field | Type | Description |
|-------|------|-------------|
| `mediaType` | string | Media type (e.g. application/json) |
| `name` | string | Alias for mediaType |
| `hasSchema` | bool | Has associated schema |
| `hasEncoding` | bool | Has encoding map |
| `hasExample` | bool | Has example or examples |
| `statusCode` | string | Status code (if from a response) |
| `operation` | string | Source operation name |

### Header Fields

Produced by the `headers` navigation stage.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Header name |
| `description` | string | Header description |
| `required` | bool | Required flag |
| `deprecated` | bool | Deprecated flag |
| `hasSchema` | bool | Has associated schema |
| `statusCode` | string | Status code of parent response |
| `operation` | string | Source operation name |

## Expressions

oq supports a rich expression language used in `where()`, `let`, and `if-then-else`:

```
depth > 5
type == "object"
name matches "Error.*"
propertyCount > 3 and not isComponent
has(oneOf) and not has(discriminator)
(depth > 10 or unionWidth > 5) and isComponent
name // "unnamed"                              # alternative: fallback if null/falsy
name default "unnamed"                         # same as above (alias)
if isComponent then depth > 3 else true end   # conditional
"prefix_\(name)"                               # string interpolation
```

### Operators

| Operator | Description |
|----------|-------------|
| `==`, `!=`, `>`, `<`, `>=`, `<=` | Comparison |
| `and`, `or`, `not` | Logical |
| `//` (or `default`) | Alternative (returns left if truthy, else right) |
| `has(field)` | True if field is non-null/non-zero |
| `matches "regex"` | Regex match |
| `if cond then a else b end` | Conditional (elif supported) |
| `\(expr)` | String interpolation inside `"..."` |

### Variables

Use `let` to bind values for use in later stages:

```
schemas | where(name == "Pet") | let $pet = name | refs-out | where(name != $pet)
```

## Output Formats

Use `--format` flag or inline `format` stage:

```bash
openapi spec query 'schemas | count' spec.yaml --format json
openapi spec query 'schemas | take(5) | format(markdown)' spec.yaml
```

| Format | Description |
|--------|-------------|
| `table` | Aligned columns (default) |
| `json` | JSON array |
| `markdown` | Markdown table |
| `toon` | [TOON](https://github.com/toon-format/toon) tabular format |

## Examples

```bash
# Wide union trees
schemas | where(unionWidth > 0) | sort-by(unionWidth, desc) | take(10)

# Central schemas (most referenced)
schemas | where(isComponent) | sort-by(inDegree, desc) | take(10) | select name, inDegree

# Operation sprawl
operations | sort-by(schemaCount, desc) | take(10) | select name, schemaCount

# Circular references
schemas | where(isCircular) | select name, path

# Shortest path between schemas
schemas | path(Pet, Address) | select name

# Walk an operation to connected schemas and back to operations
operations | where(name == "GET /users") | to-schemas | to-operations | select name, method, path

# Explain query plan
schemas | where(isComponent) | where(depth > 5) | sort-by(depth, desc) | explain

# Regex filter
schemas | where(name matches "Error.*") | select name, path

# Group by type
schemas | group-by(type)

# Edge annotations — how does Pet reference other schemas?
schemas | where(isComponent) | where(name == "Pet") | refs-out | select name, via, key, from

# Blast radius — what breaks if Error changes?
schemas | where(isComponent) | where(name == "Error") | blast-radius | length

# 2-hop neighborhood
schemas | where(isComponent) | where(name == "Pet") | neighbors(2) | select name

# Orphaned schemas
schemas | where(isComponent) | orphans | select name

# Leaf nodes
schemas | where(isComponent) | leaves | select name, inDegree

# Detect cycles
schemas | cycles

# Discover clusters
schemas | where(isComponent) | clusters

# Cross-tag schemas
schemas | cross-tag | select name, tagCount

# Schemas shared across all operations
operations | shared-refs | select name, opCount

# Variable binding — find Pet's refs-out schemas (excluding Pet itself)
schemas | where(name == "Pet") | let $pet = name | refs-out | where(name != $pet) | select name

# User-defined functions
def hot: where(inDegree > 10);
def impact($name): where(name == $name) | blast-radius;
schemas | where(isComponent) | hot | select name, inDegree

# Alternative operator — fallback for missing values
schemas | where(name // "unnamed" != "unnamed") | select name

# --- Navigation examples ---

# List all parameters for a specific operation
operations | where(name == "GET /pets") | parameters | select name, in, required

# Find operations with required query parameters
operations | parameters | where(in == "query" and required) | select name, operation

# Inspect responses for an operation
operations | where(name == "GET /pets") | responses | select statusCode, description

# Drill into content types of a response
operations | where(name == "GET /pets") | responses | where(statusCode == "200") | content-types | select mediaType, hasSchema

# Extract schemas from content types
operations | where(name == "GET /pets") | responses | content-types | to-schema | select name, type

# List response headers
operations | responses | where(statusCode == "200") | headers | select name, required, operation

# Navigate from parameter back to its operation
operations | parameters | where(name == "limit") | operation | select name, method, path

# Request body content types
operations | where(method == "post") | request-body | content-types | select mediaType, hasSchema, operation

# Extract raw YAML for a schema
schemas | where(name == "Pet") | to-yaml
```

## CLI Reference

```bash
# Run query-reference for the full language reference
openapi spec query-reference

# Inline query
openapi spec query '<query>' <spec-file>

# Query from file
openapi spec query -f query.oq <spec-file>

# With output format
openapi spec query '<query>' <spec-file> --format json

# From stdin
cat spec.yaml | openapi spec query '<query>'
```
