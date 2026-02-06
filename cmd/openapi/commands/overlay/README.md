# Overlay Commands

Commands for working with OpenAPI Overlays.

OpenAPI Overlays provide a way to modify OpenAPI and Arazzo specifications without directly editing the original files. This is useful for adding vendor-specific extensions, modifying specifications for different environments, and applying transformations to third-party APIs.

## Table of Contents

- [Table of Contents](#table-of-contents)
- [Available Commands](#available-commands)
  - [`apply`](#apply)
  - [`validate`](#validate)
  - [`compare`](#compare)
- [What are OpenAPI Overlays?](#what-are-openapi-overlays)
  - [Example Overlay](#example-overlay)
- [Common Use Cases](#common-use-cases)
- [Overlay Operations](#overlay-operations)
- [Common Options](#common-options)
- [Output Formats](#output-formats)
- [Examples](#examples)
  - [Basic Workflow](#basic-workflow)
  - [Environment-Specific Modifications](#environment-specific-modifications)
  - [Integration with Other Commands](#integration-with-other-commands)

## Available Commands

### `apply`

Apply an overlay to an OpenAPI specification.

```bash
# Apply overlay to a specification (positional arguments)
openapi overlay apply overlay.yaml spec.yaml

# Apply overlay to a specification (flags)
openapi overlay apply --overlay overlay.yaml --schema spec.yaml

# Apply overlay with output to file
openapi overlay apply --overlay overlay.yaml --schema spec.yaml --out modified-spec.yaml

# Apply overlay when overlay has extends key set
openapi overlay apply overlay.yaml
# or
openapi overlay apply --overlay overlay.yaml
```

Features:

- Applies overlay transformations to OpenAPI specifications
- Supports all OpenAPI Overlay Specification operations
- Handles complex nested modifications
- Preserves original document structure where not modified
- Supports both positional arguments and explicit flags

### `validate`

Validate an overlay file for compliance with the OpenAPI Overlay Specification.

```bash
# Validate an overlay file (positional argument)
openapi overlay validate overlay.yaml

# Validate an overlay file (flag)
openapi overlay validate --overlay overlay.yaml

# Validate with verbose output
openapi overlay validate -v overlay.yaml
```

This command checks for:

- Structural validity according to the OpenAPI Overlay Specification
- Required fields and valid values
- Proper overlay operation syntax
- Target path validity

Note: This validates the overlay file structure itself, not whether it will apply correctly to a specific OpenAPI specification.

### `compare`

Generate an OpenAPI Overlay specification from two input files.

```bash
# Generate overlay from two specifications (positional arguments)
openapi overlay compare spec1.yaml spec2.yaml

# Generate overlay from two specifications (flags)
openapi overlay compare --before spec1.yaml --after spec2.yaml

# Generate overlay with output to file
openapi overlay compare --before spec1.yaml --after spec2.yaml --out overlay.yaml
```

Features:

- Automatically detects differences between specifications
- Generates overlay operations for all changes
- Provides diagnostic output showing detected changes
- Creates overlay files that can recreate the transformation
- Supports both positional arguments and explicit flags

## What are OpenAPI Overlays?

OpenAPI Overlays are documents that describe modifications to be applied to OpenAPI specifications. They allow you to:

- **Add vendor extensions** without modifying the original spec
- **Modify specifications** for different environments (dev, staging, prod)
- **Apply transformations** to third-party APIs you don't control
- **Version control changes** separately from the base specification

### Example Overlay

```yaml
overlay: 1.0.0
info:
  title: Add API Key Authentication
  version: 1.0.0
actions:
  - target: "$.components"
    update:
      securitySchemes:
        ApiKeyAuth:
          type: apiKey
          in: header
          name: X-API-Key
  - target: "$.security"
    update:
      - ApiKeyAuth: []
```

## Common Use Cases

**Environment Configuration**: Different server URLs, authentication methods per environment
**Vendor Extensions**: Add custom extensions without modifying the original specification
**API Customization**: Modify third-party API specifications for your specific needs
**Documentation Enhancement**: Add examples, descriptions, or additional metadata
**Security Modifications**: Add or modify authentication and authorization schemes

## Overlay Operations

OpenAPI Overlays support several types of operations:

- **Update**: Merge new content with existing content
- **Remove**: Delete specific elements from the specification
- **Replace**: Completely replace existing content with new content

## Common Options

All commands support these common options:

- `-h, --help`: Show help for the command
- `-v, --verbose`: Enable verbose output (global flag)

Command-specific flags:

**apply command:**

- `--overlay`: Path to the overlay file (alternative to positional argument)
- `--schema`: Path to the OpenAPI specification (alternative to positional argument)
- `-o, --out`: Output file path (optional, defaults to stdout)

**validate command:**

- `--overlay`: Path to the overlay file (alternative to positional argument)

**compare command:**

- `--before`: Path to the first (before) specification file (alternative to positional argument)
- `--after`: Path to the second (after) specification file (alternative to positional argument)
- `-o, --out`: Output file path (optional, defaults to stdout)

## Output Formats

All commands work with both YAML and JSON input files, but always output YAML at this time. The tools preserve the structure and formatting of the original documents where possible.

## Examples

### Basic Workflow

```bash
# Create an overlay by comparing two specs (using flags)
openapi overlay compare --before original.yaml --after modified.yaml --out changes.overlay.yaml

# Or using positional arguments
openapi overlay compare original.yaml modified.yaml --out changes.overlay.yaml

# Validate the generated overlay
openapi overlay validate changes.overlay.yaml

# Apply the overlay to the original spec (using flags)
openapi overlay apply --overlay changes.overlay.yaml --schema original.yaml --out final.yaml

# Or using positional arguments
openapi overlay apply changes.overlay.yaml original.yaml --out final.yaml
```

### Environment-Specific Modifications

```bash
# Apply production overlay (using flags)
openapi overlay apply --overlay prod.overlay.yaml --schema base-spec.yaml --out prod-spec.yaml

# Or using positional arguments
openapi overlay apply prod.overlay.yaml base-spec.yaml --out prod-spec.yaml

# Apply development overlay
openapi overlay apply dev.overlay.yaml base-spec.yaml --out dev-spec.yaml
```

### Integration with Other Commands

```bash
# Validate base spec, apply overlay, then validate result
openapi spec validate ./base-spec.yaml
openapi overlay apply ./modifications.yaml ./base-spec.yaml --out ./modified-spec.yaml
openapi spec validate ./modified-spec.yaml
