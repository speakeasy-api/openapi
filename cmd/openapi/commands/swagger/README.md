# Swagger Commands

Commands for working with Swagger 2.0 (OpenAPI v2) specifications.

Swagger 2.0 documents describe REST APIs prior to OpenAPI 3.x. These commands help you validate and upgrade Swagger documents.

## Table of Contents

- [Table of Contents](#table-of-contents)
- [Available Commands](#available-commands)
  - [`validate`](#validate)
  - [`upgrade`](#upgrade)
- [What is Swagger 2.0?](#what-is-swagger-20)
- [Common Options](#common-options)
- [Output Formats](#output-formats)
- [Examples](#examples)
  - [Validate a Swagger document](#validate-a-swagger-document)
  - [Upgrade Swagger to OpenAPI 3.0](#upgrade-swagger-to-openapi-30)
  - [In-place upgrade](#in-place-upgrade)
  - [Pipe-friendly usage](#pipe-friendly-usage)

## Available Commands

### `validate`

Validate a Swagger 2.0 (OpenAPI v2) specification document for compliance.

```bash
openapi swagger validate <file>
```

This command checks for:

- Structural validity according to the Swagger 2.0 Specification
- Required fields and proper data types
- Reference resolution and consistency
- Schema validation rules

Exits with a non-zero status code when validation fails.

### `upgrade`

Convert a Swagger 2.0 document to OpenAPI 3.0 (3.0.0).

```bash
openapi swagger upgrade <input-file> [output-file]
```

The upgrade process includes:

- Converting host/basePath/schemes to `servers`
- Transforming parameters, request bodies, and responses to OAS3 structures
- Mapping `definitions` to `components.schemas`
- Migrating `securityDefinitions` to `components.securitySchemes`
- Rewriting `$ref` targets from `#/definitions/...` to `#/components/schemas/...`

Behavior:

- If no `output-file` is provided, upgraded output is written to stdout (pipe-friendly)
- If `output-file` is provided, writes the upgraded document to that file
- If `--write`/`-w` is provided, upgrades in-place (overwrites the input file)

## What is Swagger 2.0?

Swagger 2.0 is an older version of the API description format now standardized as OpenAPI 3.x. This CLI supports validating Swagger 2.0 specs and upgrading them to OpenAPI 3.0 for compatibility with modern tooling and features.

## Common Options

All commands support these common options:

- `-h, --help`: Show help for the command
- `-v, --verbose`: Enable verbose output (global flag)

Upgrade-specific options:

- `-w, --write`: Write result in-place to input file (overwrites the input)

## Output Formats

- Input files may be YAML or JSON
- Output respects YAML/JSON based on the marshaller and target file extension (when writing to a file)
- Stdout output is designed to be pipe-friendly

## Examples

### Validate a Swagger document

```bash
# Validate a JSON Swagger document
openapi swagger validate ./api.swagger.json

# Validate a YAML Swagger document
openapi swagger validate ./api.swagger.yaml
```

### Upgrade Swagger to OpenAPI 3.0

```bash
# Upgrade and write to stdout
openapi swagger upgrade ./api.swagger.yaml

# Upgrade and write to a specific file
openapi swagger upgrade ./api.swagger.yaml ./openapi.yaml
```

### In-place upgrade

```bash
# Overwrite the input file with the upgraded OpenAPI 3.0 document
openapi swagger upgrade -w ./api.swagger.yaml
```

### Pipe-friendly usage

```bash
# Upgrade and then validate with the OpenAPI validator
openapi swagger upgrade ./api.swagger.yaml | openapi spec validate -

# Upgrade and bundle
openapi swagger upgrade ./api.swagger.yaml | openapi spec bundle - ./openapi-bundled.yaml
