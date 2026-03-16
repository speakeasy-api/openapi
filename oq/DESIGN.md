# oq Navigation Model Overhaul

## Motivation

The current query language can only answer questions about schemas and operations.
It cannot answer questions like:

- Which operations serve `text/event-stream` responses?
- Which parameters are deprecated across the API?
- What content types does a specific endpoint accept?
- Which responses have no content body?

The graph builds operation→schema edges but **discards** content-type, parameter,
response, and header information in the process. These are first-class OpenAPI
constructs that users need to query.

## Design Principles

1. **Two sources, many stages.** `operations` and `schemas` are the only pipeline
   entry points. Everything else is reached by navigation stages.
2. **Navigation over filtering.** `schemas.components` and `schemas.inline` are
   removed. Use `where(isComponent)` / `where(isInline)` instead.
3. **Context propagation.** Navigation stages propagate parent context as fields
   on child rows. A content-type row carries `statusCode` from its response and
   `op_idx` from its operation — no special lineage system needed.
4. **Position disambiguates.** `schemas` as the first token = source (all schemas).
   `operations | schemas` = navigation stage. Same word, different meaning based
   on position — same as current behavior.

## Breaking Changes

| Change | Before | After |
|--------|--------|-------|
| `schemas.components` | Component schemas | Removed — use `schemas \| where(isComponent)` |
| `schemas.inline` | Inline schemas | Removed — use `schemas \| where(isInline)` |

Note: `schemas` source continues to return **all** schemas (components + inline).
This preserves the ability to query the full schema set. The sub-sources are
removed because they're just `where()` predicates.

## New Row Types

### ParameterRow

Produced by: `operations | parameters`

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Parameter name |
| `in` | string | Location: path, query, header, cookie |
| `required` | bool | Whether required |
| `deprecated` | bool | Whether deprecated |
| `description` | string | Parameter description |
| `style` | string | Serialization style |
| `explode` | bool | Whether to explode |
| `hasSchema` | bool | Whether a schema is defined |
| `allowEmptyValue` | bool | Whether empty values are allowed |
| `allowReserved` | bool | Whether reserved chars are allowed |
| `operation` | string | Source operation name (inherited) |

### ResponseRow

Produced by: `operations | responses`

| Field | Type | Description |
|-------|------|-------------|
| `statusCode` | string | "200", "404", "default", etc. |
| `description` | string | Response description |
| `contentTypeCount` | int | Number of media types |
| `headerCount` | int | Number of headers |
| `linkCount` | int | Number of links |
| `hasContent` | bool | Whether content is defined |
| `operation` | string | Source operation name (inherited) |

### RequestBodyRow

Produced by: `operations | request-body`

| Field | Type | Description |
|-------|------|-------------|
| `description` | string | Request body description |
| `required` | bool | Whether required |
| `contentTypeCount` | int | Number of media types |
| `operation` | string | Source operation name (inherited) |

### ContentTypeRow

Produced by: `responses | content-types` or `request-body | content-types`

| Field | Type | Description |
|-------|------|-------------|
| `mediaType` | string | "application/json", "text/event-stream", etc. |
| `hasSchema` | bool | Whether a schema is defined |
| `hasEncoding` | bool | Whether encoding is defined |
| `hasExample` | bool | Whether examples are defined |
| `statusCode` | string | Response status code (propagated from parent response, empty for request body) |
| `operation` | string | Source operation name (inherited) |

### HeaderRow

Produced by: `responses | headers`

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Header name |
| `description` | string | Header description |
| `required` | bool | Whether required |
| `deprecated` | bool | Whether deprecated |
| `hasSchema` | bool | Whether a schema is defined |
| `statusCode` | string | Response status code (propagated from parent response) |
| `operation` | string | Source operation name (inherited) |

## Navigation Map

Shows which stages are valid from each row type:

```
operations ──┬── parameters ────── schema
             ├── responses ──┬── content-types ── schema
             │               └── headers ──────── schema
             ├── request-body ── content-types ── schema
             └── schemas ──────── (existing graph traversal)

schemas ─────┬── refs(out), refs(in), reachable, ancestors
             ├── properties, members, items
             ├── ops
             └── (all existing schema traversal stages)
```

### Back-navigation

Every non-schema row carries the index of the operation it originated from.

- `operation` stage: from any row type with an OpIdx → yields the OperationResult
- Works on: parameters, responses, request-body, content-types, headers
- Enables: `operations | responses | content-types | where(...) | operation | unique`

### Schema Resolution

- `schema` stage (singular): extracts the schema from a parameter, content-type,
  or header row. Yields a SchemaResult row. If the schema exists in the graph
  (via pointer lookup), uses the graph node. Otherwise yields nothing.
- This bridges the navigational model back into the existing graph traversal
  system — once you have a schema row, all existing stages work.
- `schema` is distinct from `schemas` — singular extracts one schema from a
  navigational row, plural navigates operations→all reachable schemas.

## Example Queries (New Capabilities)

```bash
# Operations serving SSE responses
operations | responses | content-types | where(mediaType == "text/event-stream") | operation | unique

# All content types used across the API
operations | responses | content-types | select mediaType | unique | sort-by(mediaType)

# Operations with deprecated parameters
operations | parameters | where(deprecated) | operation | unique

# Cookie parameters (potential security review)
operations | parameters | where(in == "cookie") | select name, in, operation

# Responses without content bodies
operations | responses | where(not hasContent) | select statusCode, description, operation

# Schema for a specific content type
operations | where(name == "createUser") | request-body | content-types | where(mediaType == "application/json") | to-schema

# Headers on error responses
operations | responses | where(statusCode matches "^[45]") | headers | select name, required

# Operations that accept multipart uploads
operations | request-body | content-types | where(mediaType matches "multipart/") | operation | unique

# Content-types on 200 responses only
operations | responses | content-types | where(statusCode == "200") | select mediaType, operation
```

## Emit Attribution Fix

Separate from the navigation overhaul: `emit` should use `path` instead of `name`
as the YAML wrapper key. This gives full JSON pointer attribution:

```yaml
# Before
/properties/vault_url:
    type: string
    format: uri

# After
#/components/schemas/VaultConfig/properties/vault_url:
    type: string
    format: uri
```

## Implementation Strategy

### Row Struct Extension

Add typed fields for navigation context (matching existing pattern of typed fields):

```go
type Row struct {
    Kind      ResultKind
    SchemaIdx int
    OpIdx     int

    // Edge annotations (existing)
    Via, Key, From string

    // Group annotations (existing)
    GroupKey   string
    GroupCount int
    GroupNames []string

    // Navigation objects (new) — one is set based on Kind
    Parameter   *openapi.Parameter
    Response    *openapi.Response
    RequestBody *openapi.RequestBody
    MediaType   *openapi.MediaType
    Header      *openapi.Header

    // Propagated context (new)
    StatusCode    string // propagated from response to content-types/headers
    MediaTypeName string // the media type key (e.g., "application/json")
    HeaderName    string // the header name
    ParamName     string // parameter name
    SourceOpIdx   int    // operation this row originated from (-1 if N/A)
}
```

### No Graph Changes Required

The new navigation stages work directly with the `Operation` object stored on
`OperationNode`. No changes to the graph package are needed.

The `schema` stage needs to resolve schema pointers back to graph nodes. Add a
public `SchemaByPtr(*oas3.JSONSchemaReferenceable) (NodeID, bool)` method to
`SchemaGraph`.

### Phases

**Phase 1: Foundation**
- Add new ResultKind constants
- Extend Row struct with typed navigation fields
- Add `fieldValue` cases for new row types
- Add `defaultFieldsForKind` for new row types
- Remove `schemas.components` and `schemas.inline` sources
- Fix `emit` to use `path` as YAML key

**Phase 2: Operation Navigation**
- `parameters` stage
- `responses` stage
- `request-body` stage
- `operation` back-navigation stage
- `SchemaByPtr` method on graph

**Phase 3: Deep Navigation**
- `content-types` stage (from responses and request-body)
- `headers` stage (from responses)
- `schema` (singular) stage (from parameters, content-types, headers)

**Phase 4: Cleanup**
- Update query reference docs
- Update oq/README.md
- Update CLI README
- Remove dead code from old schemas.components/schemas.inline paths

## What Gets Removed

- `schemas.components` source
- `schemas.inline` source
- Tests for removed sources (update to use `schemas | where(isComponent)` etc.)

## What Does NOT Change

- `schemas` source (still returns all schemas)
- All existing schema traversal stages
- All existing expression language features
- All existing filter/transform stages
- The module/def system
- Output formats (table, json, markdown, toon)
- The graph package (except adding SchemaByPtr)

## Deferred (Future Work)

- **Reverse navigation** (`usages` stage): schema → content-types/parameters that
  reference it. Requires graph changes to store usage context during construction.
- **Webhooks source**: OAS 3.1 webhooks as a third entry point or merged into
  operations with an `isWebhook` field.
- **Security schemes**: queryable security requirements on operations.
- **Links**: response link objects as a navigable construct.
