# oq Query Implementation Audit

Tested against:
- `/tmp/complex-openapi.yaml` (2157 lines, 10 operations, 108 component schemas)
- GitHub REST API spec (6.5MB, 324 operations, 819 component schemas, 36K total schemas)
- CrowdStrike API spec (7.8MB, 1223 operations, 49K schemas, 10K content-types)

## Bugs

### B1: `unique` doesn't work after `pick` on navigation rows

**Severity:** High — fundamental usability issue — **FIXED**

`unique` deduplicates by `rowKey()`, which includes the full context (OpIdx,
StatusCode, MediaTypeName). After `pick media_type`, the user expects dedup by
displayed value, but `rowKey` still uses the full identity.

```
operations | responses | content-types | pick media_type | unique
→ Shows 638 rows (all "application/json" duplicated) instead of ~5 distinct types
```

This affects all row types, not just nav rows. `unique` has always been
identity-based, but with navigation rows producing many rows with the same
logical value, it becomes a visible problem.

**Fix:** `unique` should deduplicate by projected fields when `pick` is set,
falling back to `rowKey` when no projection is active.

---

### B2: `schema` stage returns `$ref` wrapper nodes, not resolved components

**Severity:** High — the bridge between navigation and graph is broken for `$ref` schemas — **FIXED**

When a content-type's schema is `{"$ref": "#/components/schemas/Foo"}`, the
`schema` stage returns the inline `$ref` wrapper node, not the `Foo` component.
The wrapper has `has_ref=true`, `is_component=false`, and a path like
`/paths/~1entities/get/responses/200/content/application~1json/schema`.

```
operations | first(1) | responses | first(1) | content-types | schema
→ Returns the $ref wrapper, not PaginatedEntityList
```

The user must do `| schema | refs-out` to get the actual component, which is
unintuitive and defeats the purpose of the `schema` stage.

**Fix:** `execSchema` should call `resolveRefTarget()` (already exists in
exec.go) to follow `$ref` edges to the actual component schema.

---

### B3: `security` stage ignores global security

**Severity:** Medium — misses the primary security model for many APIs — **FIXED**

The `security` stage only reads `op.Operation.Security` (per-operation).
Many APIs (GitHub, Stripe, etc.) define security at the root level:

```yaml
security:
  - bearerAuth: []
```

Operations without explicit security inherit the global default. Currently:
```
operations | first(1) | security | length
→ 0 (wrong — should show the inherited global security)
```

**Fix:** When `op.Operation.Security` is nil (not empty — nil means "inherit"),
fall back to `g.Index.Doc.GetSecurity()`. An explicit empty array (`security: []`)
means "no security" and should correctly return 0.

---

### B4: `components.responses` uses `StatusCode` field for component key name

**Severity:** Low — semantically misleading field name — **FIXED**

Component responses are keyed by name (e.g., "NotFound", "ValidationError"),
not by status code. The `StatusCode` field is repurposed to store the component
key name, which creates confusion:

```
components.responses | pick name, status_code
→ name=NotFound, status_code=NotFound  (status_code is meaningless here)
```

**Fix:** Add a dedicated `ComponentName` field to Row, or use `ParamName` as a
generic component key field. The `name` field alias already works correctly;
it's just that `status_code` leaks the wrong semantics.

---

## Design Issues

### D1: Navigation stages silently produce empty results on wrong row types

**Severity:** Low — defensive but confusing

```
schemas | parameters → (empty)
operations | headers → (empty)  (should work on responses, not operations)
```

These silently return empty results instead of erroring. This is arguably correct
(same as jq behavior — filters that don't match produce nothing), but can be
confusing when debugging a query.

**Recommendation:** Keep silent behavior (consistent with pipeline philosophy)
but consider adding a `--strict` flag or a lint mode that warns about
type-incompatible stage chains.

---

### D2: `group_by` on nav rows produces unhelpful `names` values

**Severity:** Low — cosmetic — **FIXED**

```
operations | responses | group_by(status_code)
→ 200: count=7 names=[200, 200, 200, 200, 200, ...]
```

`names` shows the `name` field of each row in the group. For ResponseResult,
`name` aliases to `status_code`, so you get the group key repeated. The
meaningful value would be the operation name.

**Fix:** Consider using the `operation` field instead of `name` for nav row
group names, or making the grouped field configurable.

---

### D3: `emit` key for response rows is just the status code

**Severity:** Low — limited usefulness — **FIXED**

```
operations | first(1) | responses | first(1) | emit
→ 200:
→     description: ...
```

The key "200" gives no context about which operation this response belongs to.
For schema rows, `path` provides full attribution. Response rows don't have a
`path` field, so the fallback is `name` (= status_code).

**Fix:** For nav rows, `emit` could construct a compound key from context fields,
e.g., `listEntities/200` or `GET /entities → 200`.

---

### D4: Parameter schema from `schema` stage returns the schema's own path, not the parameter's

**Severity:** Low — confusing but correct

```
operations | parameters | first(1) | schema | pick name
→ /schema
```

The parameter's schema node has name `/schema` (its JSON pointer relative to
the parameter). This is technically correct but unhelpful. The user would
prefer to see the parameter name or the schema type.

---

## Performance

### P1: Query execution is fast; document parsing dominates

All queries on all spec sizes complete in ~3.2-3.5s. The parsing/graph-build
step is ~3.2s regardless of query complexity. The actual query execution adds
<200ms even for 49K schemas or 10K content-types.

**No performance issues detected at current scale.**

### P2: No streaming — entire result materialized before output

For very large result sets (e.g., 49K schemas), all rows are materialized in
memory before formatting. This is fine for current use but could become an
issue for streaming/export use cases.

---

## Missing Features

### M1: No `has_security` field on operations

Cannot directly filter operations by whether they have security requirements:
```
operations | select(has_security)  → field doesn't exist
```

**Fix:** Add `has_security` and `security_count` (already partially exists) to
operation fields, accounting for global security inheritance.

### M2: No way to query global security requirements directly

The global `security` array can only be reached by checking operations that
inherit it. There's no `security` source.

### M3: Component request-body `name` field is generic

```
components.request-bodies | pick name
→ name=request-body (hardcoded)
```

Should show the component key name instead.

---

## What Works Well

- **All format modes** (table, json, markdown, toon) work correctly with nav rows
- **`select()` filtering** works on all nav row fields
- **`sort_by()`** works on nav row fields
- **`operation` back-navigation** correctly deduplicates and returns source operations
- **`content-types` from both responses and request-body** works correctly
- **`schema` stage** correctly bridges to the graph (aside from B2 `$ref` issue)
- **Security on CrowdStrike** (per-operation OAuth2 with scopes) works correctly
- **Performance** is excellent even on very large specs (49K schemas in <3.5s)
- **`emit` on all row types** now works (parameters, responses, content-types)
- **`components.*` sources** correctly iterate named component definitions
- **`explain`** correctly describes navigation stages
