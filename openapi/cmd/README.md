# OpenAPI Commands

Commands for working with OpenAPI specifications.

OpenAPI specifications define REST APIs in a standard format. These commands help you validate, transform, and work with OpenAPI documents.

## Table of Contents

- [Table of Contents](#table-of-contents)
- [Available Commands](#available-commands)
  - [`validate`](#validate)
  - [`upgrade`](#upgrade)
  - [`inline`](#inline)
  - [`bundle`](#bundle)
    - [Bundle vs Inline](#bundle-vs-inline)
  - [`join`](#join)
  - [`bootstrap`](#bootstrap)
- [Common Options](#common-options)
- [Output Formats](#output-formats)
- [Examples](#examples)
  - [Validation Workflow](#validation-workflow)
  - [Processing Pipeline](#processing-pipeline)

## Available Commands

### `validate`

Validate an OpenAPI specification document for compliance with the OpenAPI Specification.

```bash
# Validate a specification file
openapi spec validate ./spec.yaml

# Validate with verbose output
openapi spec validate -v ./spec.yaml
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
openapi spec upgrade ./spec.yaml

# Upgrade to specific file
openapi spec upgrade ./spec.yaml ./upgraded-spec.yaml

# Upgrade in-place
openapi spec upgrade -w ./spec.yaml

# Upgrade with specific target version
openapi spec upgrade --version 3.1.0 ./spec.yaml
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
openapi spec inline ./spec-with-refs.yaml

# Inline to specific file
openapi spec inline ./spec.yaml ./inlined-spec.yaml

# Inline in-place
openapi spec inline -w ./spec.yaml
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
openapi spec bundle ./spec-with-refs.yaml

# Bundle to specific file with filepath naming (default)
openapi spec bundle ./spec.yaml ./bundled-spec.yaml

# Bundle in-place with counter naming
openapi spec bundle -w --naming counter ./spec.yaml

# Bundle with filepath naming (explicit)
openapi spec bundle --naming filepath ./spec.yaml ./bundled.yaml
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

#### Bundle vs Inline

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

### `join`

Join multiple OpenAPI specifications into a single document.

```bash
# Join specifications to stdout
openapi spec join ./main.yaml ./additional.yaml

# Join specifications to specific file
openapi spec join ./main.yaml ./additional.yaml ./joined-spec.yaml

# Join in-place (modifies the first file)
openapi spec join -w ./main.yaml ./additional.yaml

# Join with conflict resolution strategy
openapi spec join --strategy merge ./main.yaml ./additional.yaml
```

Features:

- Combines multiple OpenAPI specifications into one
- Handles conflicts between specifications intelligently
- Merges paths, components, and other sections
- Preserves all valid OpenAPI structure and references

### `bootstrap`

Create a new OpenAPI document with best practice examples.

```bash
# Create bootstrap document and output to stdout
openapi spec bootstrap

# Create bootstrap document and save to file
openapi spec bootstrap ./my-api.yaml

# Create bootstrap document in current directory
openapi spec bootstrap ./openapi.yaml
```

What bootstrap creates:

- Complete OpenAPI specification template with comprehensive examples
- Proper document structure and metadata (info, servers, tags)
- Example operations with request/response definitions
- Reusable components (schemas, responses, security schemes)
- Reference usage ($ref) for component reuse
- Security scheme definitions (API key authentication)
- Comprehensive schema examples with validation rules

The generated document serves as both a template for new APIs and a learning resource for OpenAPI best practices.

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
openapi spec validate ./spec.yaml

# Upgrade if needed
openapi spec upgrade ./spec.yaml ./spec-v3.1.yaml

# Bundle external references
openapi spec bundle ./spec-v3.1.yaml ./spec-bundled.yaml

# Final validation
openapi spec validate ./spec-bundled.yaml
```

### Processing Pipeline

```bash
# Create a processing pipeline
openapi spec bundle ./spec.yaml | \
openapi spec upgrade | \
openapi spec validate
