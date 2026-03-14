# oq — OpenAPI Query Language

`oq` is a pipeline query language for exploring OpenAPI schema reference graphs. It lets you ask structural and semantic questions about schemas and operations at the command line.

## Quick Start

```bash
# Count all schemas
openapi spec query 'schemas | count' petstore.yaml

# Top 10 deepest component schemas
openapi spec query 'schemas | select(is_component) | sort_by(depth; desc) | first(10) | pick name, depth' petstore.yaml

# Dead components (unreferenced)
openapi spec query 'schemas | select(is_component) | select(in_degree == 0) | pick name' petstore.yaml
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
| `refs-out` | Direct outgoing references (with edge annotations) |
| `refs-in` | Direct incoming references (with edge annotations) |
| `reachable` | Transitive closure of outgoing refs |
| `ancestors` | Transitive closure of incoming refs |
| `properties` | Property sub-schemas (with edge annotations) |
| `union-members` | allOf/oneOf/anyOf children (with edge annotations) |
| `items` | Array items schema (with edge annotations) |
| `parent` | Navigate back to source schema of edge annotations |
| `ops` | Schemas → operations |
| `schemas` | Operations → schemas |
| `path(A; B)` | Shortest path between two schemas |
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
| `schema` | Extract schema from parameter, content-type, or header |
| `operation` | Back-navigate to source operation |

### Analysis Stages

| Stage | Description |
|-------|-------------|
| `orphans` | Schemas with no incoming refs and no operation usage |
| `leaves` | Schemas with no outgoing refs (terminal nodes) |
| `cycles` | Strongly connected components (actual cycles) |
| `clusters` | Weakly connected component grouping |
| `tag-boundary` | Schemas used by operations across multiple tags |
| `shared-refs` | Schemas shared by ALL operations in result set |

### Filter & Transform Stages

| Stage | Description |
|-------|-------------|
| `select(expr)` | Filter by predicate (jq-style) |
| `pick f1, f2` | Project fields |
| `sort_by(field)` / `sort_by(field; desc)` | Sort (ascending by default) |
| `first(N)` | Limit to first N results |
| `last(N)` | Limit to last N results |
| `sample(N)` | Deterministic random sample |
| `top(N; field)` | Sort desc + take |
| `bottom(N; field)` | Sort asc + take |
| `unique` | Deduplicate |
| `group_by(field)` | Group and count |
| `length` | Count rows |
| `let $var = expr` | Bind expression result to a variable |

### Meta Stages

| Stage | Description |
|-------|-------------|
| `explain` | Print query plan |
| `fields` | List available fields |
| `format(fmt)` | Set output format (table/json/markdown/toon) |
| `emit` | Output raw YAML nodes from underlying spec objects |

The `emit` stage uses `path` (JSON pointer) as the wrapper key for each emitted node, giving full attribution to the source location in the spec.

### Function Definitions & Modules

Define reusable functions with `def` and load them from `.oq` files with `include`:

```
# Inline definitions
def hot: select(in_degree > 10);
def impact($name): select(name == $name) | blast-radius;
schemas | select(is_component) | hot | pick name, in_degree

# Load from file
include "stdlib.oq";
schemas | select(is_component) | hot | pick name, in_degree
```

Def syntax: `def name: body;` or `def name($p1; $p2): body;`
Module search paths: current directory, then `~/.config/oq/`

## Fields

### Schema Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Component name or JSON pointer |
| `type` | string | Schema type |
| `depth` | int | Max nesting depth |
| `in_degree` | int | Incoming reference count |
| `out_degree` | int | Outgoing reference count |
| `union_width` | int | Union member count |
| `property_count` | int | Property count |
| `is_component` | bool | In components/schemas |
| `is_inline` | bool | Defined inline |
| `is_circular` | bool | Part of circular reference |
| `has_ref` | bool | Has $ref |
| `hash` | string | Content hash |
| `path` | string | JSON pointer |
| `op_count` | int | Operations using this schema |
| `tag_count` | int | Distinct tags across operations |

### Operation Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | operationId or METHOD /path |
| `method` | string | HTTP method |
| `path` | string | URL path |
| `operation_id` | string | operationId |
| `schema_count` | int | Reachable schema count |
| `component_count` | int | Reachable component count |
| `tag` | string | First tag |
| `parameter_count` | int | Parameter count |
| `deprecated` | bool | Deprecated flag |
| `description` | string | Description |
| `summary` | string | Summary |

### Edge Annotation Fields

Available on rows produced by 1-hop traversal stages (`refs-out`, `refs-in`, `properties`, `union-members`, `items`). Use `parent` to navigate back to the source schema.

| Field | Type | Description |
|-------|------|-------------|
| `via` | string | Edge type: property, items, allOf, oneOf, ref, ... |
| `key` | string | Edge key: property name, array index, etc. |
| `from` | string | Source node name |

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
| `has_schema` | bool | Has associated schema |
| `allow_empty_value` | bool | Allow empty value |
| `allow_reserved` | bool | Allow reserved characters |
| `operation` | string | Source operation name |

### Response Fields

Produced by the `responses` navigation stage.

| Field | Type | Description |
|-------|------|-------------|
| `status_code` | string | HTTP status code |
| `name` | string | Alias for status_code |
| `description` | string | Response description |
| `content_type_count` | int | Number of content types |
| `header_count` | int | Number of headers |
| `link_count` | int | Number of links |
| `has_content` | bool | Has content types |
| `operation` | string | Source operation name |

### Request Body Fields

Produced by the `request-body` navigation stage.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Always "request-body" |
| `description` | string | Request body description |
| `required` | bool | Required flag |
| `content_type_count` | int | Number of content types |
| `operation` | string | Source operation name |

### Content Type Fields

Produced by the `content-types` navigation stage.

| Field | Type | Description |
|-------|------|-------------|
| `media_type` | string | Media type (e.g. application/json) |
| `name` | string | Alias for media_type |
| `has_schema` | bool | Has associated schema |
| `has_encoding` | bool | Has encoding map |
| `has_example` | bool | Has example or examples |
| `status_code` | string | Status code (if from a response) |
| `operation` | string | Source operation name |

### Header Fields

Produced by the `headers` navigation stage.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Header name |
| `description` | string | Header description |
| `required` | bool | Required flag |
| `deprecated` | bool | Deprecated flag |
| `has_schema` | bool | Has associated schema |
| `status_code` | string | Status code of parent response |
| `operation` | string | Source operation name |

## Expressions

oq supports a rich expression language used in `select()`, `let`, and `if-then-else`:

```
depth > 5
type == "object"
name matches "Error.*"
property_count > 3 and not is_component
has(oneOf) and not has(discriminator)
(depth > 10 or union_width > 5) and is_component
name // "unnamed"                              # alternative: fallback if null/falsy
if is_component then depth > 3 else true end   # conditional
"prefix_\(name)"                               # string interpolation
```

### Operators

| Operator | Description |
|----------|-------------|
| `==`, `!=`, `>`, `<`, `>=`, `<=` | Comparison |
| `and`, `or`, `not` | Logical |
| `//` | Alternative (returns left if truthy, else right) |
| `has(field)` | True if field is non-null/non-zero |
| `matches "regex"` | Regex match |
| `if cond then a else b end` | Conditional (elif supported) |
| `\(expr)` | String interpolation inside `"..."` |

### Variables

Use `let` to bind values for use in later stages:

```
schemas | select(name == "Pet") | let $pet = name | reachable | select(name != $pet)
```

## Output Formats

Use `--format` flag or inline `format` stage:

```bash
openapi spec query 'schemas | count' spec.yaml --format json
openapi spec query 'schemas | first(5) | format(markdown)' spec.yaml
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
schemas | select(union_width > 0) | sort_by(union_width; desc) | first(10)

# Central schemas (most referenced)
schemas | select(is_component) | sort_by(in_degree; desc) | first(10) | pick name, in_degree

# Operation sprawl
operations | sort_by(schema_count; desc) | first(10) | pick name, schema_count

# Circular references
schemas | select(is_circular) | pick name, path

# Shortest path between schemas
schemas | path(Pet; Address) | pick name

# Walk an operation to connected schemas and back to operations
operations | select(name == "GET /users") | schemas | ops | pick name, method, path

# Explain query plan
schemas | select(is_component) | select(depth > 5) | sort_by(depth; desc) | explain

# Regex filter
schemas | select(name matches "Error.*") | pick name, path

# Group by type
schemas | group_by(type)

# Edge annotations — how does Pet reference other schemas?
schemas | select(is_component) | select(name == "Pet") | refs-out | pick name, via, key, from

# Blast radius — what breaks if Error changes?
schemas | select(is_component) | select(name == "Error") | blast-radius | length

# 2-hop neighborhood
schemas | select(is_component) | select(name == "Pet") | neighbors(2) | pick name

# Orphaned schemas
schemas | select(is_component) | orphans | pick name

# Leaf nodes
schemas | select(is_component) | leaves | pick name, in_degree

# Detect cycles
schemas | cycles

# Discover clusters
schemas | select(is_component) | clusters

# Cross-tag schemas
schemas | tag-boundary | pick name, tag_count

# Schemas shared across all operations
operations | shared-refs | pick name, op_count

# Variable binding — find Pet's reachable schemas (excluding Pet itself)
schemas | select(name == "Pet") | let $pet = name | reachable | select(name != $pet) | pick name

# User-defined functions
def hot: select(in_degree > 10);
def impact($name): select(name == $name) | blast-radius;
schemas | select(is_component) | hot | pick name, in_degree

# Alternative operator — fallback for missing values
schemas | select(name // "unnamed" != "unnamed") | pick name

# --- Navigation examples ---

# List all parameters for a specific operation
operations | select(name == "GET /pets") | parameters | pick name, in, required

# Find operations with required query parameters
operations | parameters | select(in == "query" and required) | pick name, operation

# Inspect responses for an operation
operations | select(name == "GET /pets") | responses | pick status_code, description

# Drill into content types of a response
operations | select(name == "GET /pets") | responses | select(status_code == "200") | content-types | pick media_type, has_schema

# Extract schemas from content types
operations | select(name == "GET /pets") | responses | content-types | schema | pick name, type

# List response headers
operations | responses | select(status_code == "200") | headers | pick name, required, operation

# Navigate from parameter back to its operation
operations | parameters | select(name == "limit") | operation | pick name, method, path

# Request body content types
operations | select(method == "post") | request-body | content-types | pick media_type, has_schema, operation

# Emit raw YAML for a schema
schemas | select(name == "Pet") | emit
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
