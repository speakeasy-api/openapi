# OpenAPI Commands

Commands for working with OpenAPI specifications.

OpenAPI specifications define REST APIs in a standard format. These commands help you validate, transform, and work with OpenAPI documents.

## Available Commands

### `validate`

Validate an OpenAPI specification document for compliance with the OpenAPI Specification.

```bash
# Validate a specification file
openapi openapi validate ./spec.yaml

# Validate with verbose output
openapi openapi validate -v ./spec.yaml
```

This command checks for:

- Structural validity according to the OpenAPI Specification
- Schema compliance and consistency
- Reference resolution and validity
- Best practice recommendations

### `upgrade`

Upgrade an OpenAPI specification to the latest supported version (3.1.1).

```bash
# Upgrade to stdout
openapi openapi upgrade ./spec.yaml

# Upgrade to specific file
openapi openapi upgrade ./spec.yaml ./upgraded-spec.yaml

# Upgrade in-place
openapi openapi upgrade -w ./spec.yaml

# Upgrade with specific target version
openapi openapi upgrade --version 3.1.0 ./spec.yaml
```

Features:

- Converts OpenAPI 3.0.x specifications to 3.1.x
- Maintains backward compatibility where possible
- Updates schema formats and structures
- Preserves all custom extensions and vendor-specific content

### `inline`

Inline all references in an OpenAPI specification to create a self-contained document.

```bash
# Inline to stdout (pipe-friendly)
openapi openapi inline ./spec-with-refs.yaml

# Inline to specific file
openapi openapi inline ./spec.yaml ./inlined-spec.yaml

# Inline in-place
openapi openapi inline -w ./spec.yaml
```

What inlining does:

- Replaces all `$ref` references with their actual content
- Creates a completely self-contained document
- Removes unused components after inlining
- Handles circular references using JSON Schema `$defs`

**Before inlining:**

```yaml
paths:
  /users:
    get:
      responses:
        '200':
          $ref: "#/components/responses/UserResponse"
components:
  responses:
    UserResponse:
      description: User response
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/User"
```

**After inlining:**

```yaml
paths:
  /users:
    get:
      responses:
        '200':
          description: User response
          content:
            application/json:
              schema:
                type: object
                properties:
                  name:
                    type: string
# components section removed (unused after inlining)
```

### `bundle`

Bundle external references into the components section while preserving the reference structure.

```bash
# Bundle to stdout (pipe-friendly)
openapi openapi bundle ./spec-with-refs.yaml

# Bundle to specific file with filepath naming (default)
openapi openapi bundle ./spec.yaml ./bundled-spec.yaml

# Bundle in-place with counter naming
openapi openapi bundle -w --naming counter ./spec.yaml

# Bundle with filepath naming (explicit)
openapi openapi bundle --naming filepath ./spec.yaml ./bundled.yaml
```

**Naming Strategies:**

- `filepath` (default): Uses file path-based naming like `external_api_yaml~User` for conflicts
- `counter`: Uses counter-based suffixes like `User_1`, `User_2` for conflicts

What bundling does:

- Brings all external references into the components section
- Maintains reference structure (unlike inline which expands everything)
- Creates self-contained documents that work with reference-aware tooling
- Handles circular references and naming conflicts intelligently

**Before bundling:**

```yaml
paths:
  /users:
    get:
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: "external_api.yaml#/User"
```

**After bundling:**

```yaml
paths:
  /users:
    get:
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
components:
  schemas:
    User:
      type: object
      properties:
        id: {type: string}
        name: {type: string}
```

## Bundle vs Inline

**Use Bundle when:**

- You want a self-contained document but need to preserve references
- Your tooling works better with references than expanded content
- You want to maintain the logical structure of your API definition
- You need to prepare documents for further processing

**Use Inline when:**

- You want a completely expanded document with no references
- Your tooling doesn't support references well
- You want the simplest possible document structure
- You're creating documentation or examples

## Common Options

All commands support these common options:

- `-h, --help`: Show help for the command
- `-v, --verbose`: Enable verbose output (global flag)
- `-w, --write`: Write output back to the input file (where applicable)

## Output Formats

All commands work with both YAML and JSON input files and preserve the original format in the output. When writing to stdout (for piping), the output is optimized to be clean and parseable.

## Examples

### Validation Workflow

```bash
# Validate before processing
openapi openapi validate ./spec.yaml

# Upgrade if needed
openapi openapi upgrade ./spec.yaml ./spec-v3.1.yaml

# Bundle external references
openapi openapi bundle ./spec-v3.1.yaml ./spec-bundled.yaml

# Final validation
openapi openapi validate ./spec-bundled.yaml
```

### Processing Pipeline

```bash
# Create a processing pipeline
openapi openapi bundle ./spec.yaml | \
openapi openapi upgrade | \
openapi openapi validate
