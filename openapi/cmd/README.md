# OpenAPI Commands

Commands for working with OpenAPI specifications.

OpenAPI specifications define REST APIs in a standard format. These commands help you validate, transform, and work with OpenAPI documents.

## Table of Contents

- [Table of Contents](#table-of-contents)
- [Available Commands](#available-commands)
  - [`validate`](#validate)
  - [`upgrade`](#upgrade)
  - [`inline`](#inline)
  - [`clean`](#clean)
  - [`bundle`](#bundle)
    - [Bundle vs Inline](#bundle-vs-inline)
  - [`join`](#join)
  - [`optimize`](#optimize)
  - [`bootstrap`](#bootstrap)
  - [`localize`](#localize)
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

### `clean`

Remove unused components from an OpenAPI specification to create a cleaner, more maintainable document.

```bash
# Clean to stdout (pipe-friendly)
openapi spec clean ./spec.yaml

# Clean to specific file
openapi spec clean ./spec.yaml ./cleaned-spec.yaml

# Clean in-place
openapi spec clean -w ./spec.yaml
```

What cleaning does:

- Removes unused components from all component types (schemas, responses, parameters, etc.)
- Tracks all references throughout the document including `$ref` and security scheme name references
- Preserves all components that are actually used in the specification
- Handles complex reference patterns including circular references and nested components

**Before cleaning:**

```yaml
paths:
  /users:
    get:
      responses:
        '200':
          $ref: "#/components/responses/UserResponse"
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
    UnusedSchema:  # This will be removed
      type: object
      properties:
        id:
          type: string
  responses:
    UserResponse:
      description: User response
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/User"
    UnusedResponse:  # This will be removed
      description: Unused response
```

**After cleaning:**

```yaml
paths:
  /users:
    get:
      responses:
        '200':
          $ref: "#/components/responses/UserResponse"
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
  responses:
    UserResponse:
      description: User response
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/User"
# UnusedSchema and UnusedResponse removed
```

**Use Clean when:**

- You want to remove unused components after refactoring
- You're preparing a specification for publication or distribution
- You want to reduce document size and complexity
- You're maintaining a large specification with many components

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

### `optimize`

Optimize an OpenAPI specification by finding duplicate inline schemas and extracting them to reusable components.

```bash
# Interactive optimization (shows schemas, prompts for custom names)
openapi spec optimize ./spec.yaml

# Non-interactive optimization (auto-generated names)
openapi spec optimize ./spec.yaml --non-interactive

# Optimize to specific file
openapi spec optimize ./spec.yaml ./optimized-spec.yaml

# Optimize in-place
openapi spec optimize -w ./spec.yaml
```

What optimization does:

- Finds inline JSON schemas that appear multiple times with identical content
- Replaces duplicate inline schemas with references to newly created components
- Preserves existing component schemas (not modified or replaced)
- Only processes complex schemas (objects, enums, oneOf/allOf/anyOf, conditionals)
- Ignores simple type schemas (string, number, boolean) that don't benefit from extraction

**Interactive Mode (default):**

- Shows each duplicate schema in a beautiful formatted code block
- Displays all locations where the schema appears
- Prompts for custom component names
- Allows meaningful naming instead of auto-generated names

**Non-Interactive Mode (`--non-interactive`):**

- Uses automatically generated names based on schema content hash
- No user prompts - suitable for automation and CI/CD pipelines
- Generates names like `Schema_da0c4bbf` based on content

**Before optimization:**

```yaml
paths:
  /users:
    get:
      responses:
        '200':
          content:
            application/json:
              schema:
                type: object
                properties:
                  id: {type: integer}
                  name: {type: string}
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                id: {type: integer}
                name: {type: string}
```

**After optimization:**

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
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/User"
components:
  schemas:
    User:
      type: object
      properties:
        id: {type: integer}
        name: {type: string}
```

**Benefits of optimization:**

- Reduces document size by eliminating duplicate schema definitions
- Improves maintainability by centralizing schema definitions
- Enhances reusability by making schemas available as components
- Optimizes tooling performance with smaller, cleaner documents
- Follows OpenAPI best practices for schema organization

**Use Optimize when:**

- You have inline schemas that are duplicated across your specification
- You want to improve document maintainability and reduce redundancy
- You're preparing specifications for better tooling support
- You want to follow OpenAPI best practices for component reuse

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

### `localize`

Localize an OpenAPI specification by copying all external reference files to a target directory and creating a new version of the document with references rewritten to point to the localized files.

```bash
# Localize to a target directory with path-based naming (default)
openapi spec localize ./spec.yaml ./localized/

# Localize with counter-based naming for conflicts
openapi spec localize --naming counter ./spec.yaml ./localized/

# Localize with explicit path-based naming
openapi spec localize --naming path ./spec.yaml ./localized/
```

**Naming Strategies:**

- `path` (default): Uses file path-based naming like `schemas-address.yaml` for conflicts
- `counter`: Uses counter-based suffixes like `address_1.yaml` for conflicts

What localization does:

- Copies all external reference files to the target directory
- Creates a new version of the main document with updated references
- Leaves the original document and files completely untouched
- Creates a portable, self-contained document bundle
- Handles circular references and naming conflicts intelligently
- Supports both file-based and URL-based external references

**Before localization:**

```yaml
# main.yaml
paths:
  /users:
    get:
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: "./components.yaml#/components/schemas/User"

# components.yaml
components:
  schemas:
    User:
      properties:
        address:
          $ref: "./schemas/address.yaml#/Address"

# schemas/address.yaml
Address:
  type: object
  properties:
    street: {type: string}
```

**After localization (in target directory):**

```yaml
# localized/main.yaml
paths:
  /users:
    get:
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: "components.yaml#/components/schemas/User"

# localized/components.yaml
components:
  schemas:
    User:
      properties:
        address:
          $ref: "schemas-address.yaml#/Address"

# localized/schemas-address.yaml
Address:
  type: object
  properties:
    street: {type: string}
```

**Benefits of localization:**

- Creates portable document bundles for easy distribution
- Simplifies deployment by packaging all dependencies together
- Enables offline development without external file dependencies
- Improves version control by keeping all related files together
- Ensures all dependencies are available in CI/CD pipelines
- Facilitates documentation generation with complete file sets

**Use Localize when:**

- You need to package an API specification for distribution
- You want to create a self-contained bundle for deployment
- You're preparing specifications for offline use or air-gapped environments
- You need to ensure all dependencies are available in build pipelines
- You want to simplify file management for complex multi-file specifications
- You're creating documentation packages that include all referenced files

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
openapi spec clean | \
openapi spec upgrade | \
openapi spec validate

# Alternative: Clean after bundling to remove unused components
openapi spec bundle ./spec.yaml ./bundled.yaml
openapi spec clean ./bundled.yaml ./clean-bundled.yaml
openapi spec validate ./clean-bundled.yaml
