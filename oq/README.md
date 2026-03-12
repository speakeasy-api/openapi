# oq — OpenAPI Query Language

`oq` is a pipeline query language for exploring OpenAPI schema reference graphs. It lets you ask structural and semantic questions about schemas and operations at the command line.

## Quick Start

```bash
# Count all schemas
openapi spec query petstore.yaml 'schemas | count'

# Top 10 deepest component schemas
openapi spec query petstore.yaml 'schemas.components | sort depth desc | take 10 | select name, depth'

# Dead components (unreferenced)
openapi spec query petstore.yaml 'schemas.components | where in_degree == 0 | select name'
```

Stdin is supported:

```bash
cat spec.yaml | openapi spec query - 'schemas | count'
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
| `refs-out` | Direct outgoing references |
| `refs-in` | Direct incoming references |
| `reachable` | Transitive closure of outgoing refs |
| `ancestors` | Transitive closure of incoming refs |
| `properties` | Property sub-schemas |
| `union-members` | allOf/oneOf/anyOf children |
| `items` | Array items schema |
| `ops` | Schemas → operations |
| `schemas` | Operations → schemas |
| `path <a> <b>` | Shortest path between two schemas |

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
openapi spec query spec.yaml 'schemas | count' --format json
openapi spec query spec.yaml 'schemas | take 5 | format markdown'
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
```

## CLI Reference

```bash
# Run query-reference for the full language reference
openapi spec query-reference

# Inline query
openapi spec query <spec-file> '<query>'

# Query from file
openapi spec query <spec-file> -f query.oq

# With output format
openapi spec query <spec-file> '<query>' --format json
```
