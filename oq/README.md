# oq — OpenAPI Query Language

`oq` is a pipeline query language for exploring OpenAPI schema reference graphs. It lets you ask structural and semantic questions about schemas and operations at the command line.

## Quick Start

```bash
# Count all schemas
openapi spec query 'schemas | count' petstore.yaml

# Top 10 deepest component schemas
openapi spec query 'schemas.components | sort depth desc | take 10 | select name, depth' petstore.yaml

# Dead components (unreferenced)
openapi spec query 'schemas.components | where in_degree == 0 | select name' petstore.yaml
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
| `schemas.components` | Component schemas only |
| `schemas.inline` | Inline schemas only |
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
| `ops` | Schemas → operations |
| `schemas` | Operations → schemas |
| `path <a> <b>` | Shortest path between two schemas |
| `connected` | Full connected component (schemas + operations) |
| `blast-radius` | Ancestors + all affected operations |
| `neighbors <n>` | Bidirectional neighborhood within N hops |

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
| `where <expr>` | Filter by predicate |
| `select <fields>` | Project fields |
| `sort <field> [desc]` | Sort (ascending by default) |
| `take <n>` / `head <n>` | Limit results |
| `sample <n>` | Deterministic random sample |
| `top <n> <field>` | Sort desc + take |
| `bottom <n> <field>` | Sort asc + take |
| `unique` | Deduplicate |
| `group-by <field>` | Group and count |
| `count` | Count rows |

### Meta Stages

| Stage | Description |
|-------|-------------|
| `explain` | Print query plan |
| `fields` | List available fields |
| `format <fmt>` | Set output format (table/json/markdown/toon) |

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

Available on rows produced by 1-hop traversal stages (`refs-out`, `refs-in`, `properties`, `union-members`, `items`):

| Field | Type | Description |
|-------|------|-------------|
| `edge_kind` | string | Edge type: property, items, allOf, oneOf, ref, ... |
| `edge_label` | string | Edge label: property name, array index, etc. |
| `edge_from` | string | Source node name |

## Where Expressions

```
depth > 5
type == "object"
name matches "Error.*"
property_count > 3 and not is_component
has(oneOf) and not has(discriminator)
(depth > 10 or union_width > 5) and is_component
```

Operators: `==`, `!=`, `>`, `<`, `>=`, `<=`, `and`, `or`, `not`, `has()`, `matches()`

## Output Formats

Use `--format` flag or inline `format` stage:

```bash
openapi spec query 'schemas | count' spec.yaml --format json
openapi spec query 'schemas | take 5 | format markdown' spec.yaml
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
schemas | where union_width > 0 | sort union_width desc | take 10

# Central schemas (most referenced)
schemas.components | sort in_degree desc | take 10 | select name, in_degree

# Operation sprawl
operations | sort schema_count desc | take 10 | select name, schema_count

# Circular references
schemas | where is_circular | select name, path

# Shortest path between schemas
schemas | path "Pet" "Address" | select name

# Walk an operation to connected schemas and back to operations
operations | where name == "GET /users" | schemas | ops | select name, method, path

# Explain query plan
schemas.components | where depth > 5 | sort depth desc | explain

# Regex filter
schemas | where name matches "Error.*" | select name, path

# Group by type
schemas | group-by type

# Edge annotations — how does Pet reference other schemas?
schemas.components | where name == "Pet" | refs-out | select name, edge_kind, edge_label, edge_from

# Blast radius — what breaks if Error changes?
schemas.components | where name == "Error" | blast-radius | count

# 2-hop neighborhood
schemas.components | where name == "Pet" | neighbors 2 | select name

# Orphaned schemas
schemas.components | orphans | select name

# Leaf nodes
schemas.components | leaves | select name, in_degree

# Detect cycles
schemas | cycles

# Discover clusters
schemas.components | clusters

# Cross-tag schemas
schemas | tag-boundary | select name, tag_count

# Schemas shared across all operations
operations | shared-refs | select name, op_count
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
