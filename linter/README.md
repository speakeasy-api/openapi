# Linter Engine

This document provides an overview of the linter engine implementation.

## Architecture Overview

The linter engine is a generic, spec-agnostic framework for implementing configurable linting rules across different API specifications (OpenAPI, Arazzo, Swagger).

### Core Components

1. **Generic Linter Engine** ([`linter/`](linter/))
   - [`Linter[T]`](linter/linter.go) - Main linting engine with configuration support
   - [`Registry[T]`](linter/registry.go) - Rule registry with category management
   - [`Rule`](linter/rule.go) - Base rule interface and specialized interfaces
   - [`RuleConfig`](linter/config.go) - Per-rule configuration with severity overrides
   - [`DocumentInfo[T]`](linter/document.go) - Document + location for reference resolution
   - Format types for text and JSON output
   - Parallel rule execution for improved performance

2. **OpenAPI Linter** ([`openapi/linter/`](openapi/linter/))
   - OpenAPI-specific linter implementation
   - Rule registry with built-in rules
   - Integration with OpenAPI parser and validator

3. **Rules** ([`openapi/linter/rules/`](openapi/linter/rules/))
   - Individual linting rules (e.g., [`style-path-params`](openapi/linter/rules/path_params.go))
   - Each rule implements the [`RuleRunner[*openapi.OpenAPI]`](linter/rule.go) interface

4. **CLI Integration** ([`cmd/openapi/commands/openapi.spec/lint.go`](cmd/openapi/commands/openapi.spec/lint.go))
   - `openapi spec lint` command
   - Configuration file support (`.lint.yaml`)
   - Rule documentation generation (`--list-rules`)

## Key Features

### 1. Rule Configuration

Rules can be configured via YAML configuration file:

```yaml
extends:
  - all  # or specific rulesets like "recommended", "strict"

categories:
  style:
    enabled: true
    severity: warning

rules:
  style-path-params:
    enabled: true
    severity: error
    options:
      # Rule-specific options
```

### 2. Severity Overrides

Rules have default severities that can be overridden:
-  Fatal errors (terminate execution)
- Error severity (build failures)
- Warning severity (informational)

### 3. External Reference Resolution

Rules automatically resolve external references (HTTP URLs, file paths):

```yaml
paths:
  /users/{userId}:
    get:
      parameters:
        - $ref: "https://example.com/params/user-id.yaml"
      responses:
        '200':
          description: ok
```

The linter:
- Uses [`DocumentInfo.Location`](linter/document.go) as the base for resolving relative references
- Supports custom HTTP clients and virtual filesystems via [`LintOptions.ResolveOptions`](linter/document.go)
- Reports resolution errors as validation errors with proper severity and location

### 5. Quick Fix Suggestions

Rules can suggest fixes using [`validation.Error`](validation/validation.go) with quick fix support:

```go
validation.NewValidationErrorWithQuickFix(
    severity,
    rule,
    fmt.Errorf("path parameter {%s} is not defined", param),
    node,
    &validation.QuickFix{
        Description: "Add missing path parameter",
        Replacement: "...",
    },
)
```

## Implemented Rules

### style-path-params

Ensures path template variables (e.g., `{userId}`) have corresponding parameter definitions with `in='path'`.

**Checks:**
- All template params must have corresponding parameter definitions
- All path parameters must be used in the template
- Works with parameters at PathItem level (inherited) and Operation level (can override)
- Resolves external references to parameters

**Example:**

```yaml
# ✅ Valid
paths:
  /users/{userId}:
    get:
      parameters:
        - name: userId
          in: path
          required: true

# ❌ Invalid - missing parameter definition
paths:
  /users/{userId}:
    get:
      responses:
        '200':
          description: ok
```

## Usage

### CLI

```bash
# Lint with default configuration
openapi spec lint openapi.yaml

# Lint with custom config
openapi spec lint --config .lint.yaml openapi.yaml

# List all available rules
openapi spec lint --list-rules

# Output in JSON format
openapi spec lint --format json openapi.yaml
```

### Programmatic

```go
import (
    "context"
    "github.com/speakeasy-api/openapi/linter"
    openapiLinter "github.com/speakeasy-api/openapi/openapi/linter"
)

// Create linter with configuration
config := &linter.Config{
    Extends: []string{"all"},
}
lntr := openapiLinter.NewOpenAPILinter(config)

// Lint document
docInfo := &linter.DocumentInfo[*openapi.OpenAPI]{
    Document: doc,
    Location: "/path/to/openapi.yaml",
}
output, err := lntr.Lint(ctx, docInfo, nil, nil)
if err != nil {
    // Handle error
}

// Check results
if output.HasErrors() {
    fmt.Println(output.FormatText())
}
```

## Adding New Rules

To add a new rule:

1. **Create the rule** in [`openapi/linter/rules/`](openapi/linter/rules/)

```go
type MyRule struct{}

func (r *MyRule) ID() string { return "style-my-rule" }
func (r *MyRule) Category() string { return "style" }
func (r *MyRule) Description() string { return "..." }
func (r *MyRule) Link() string { return "..." }
func (r *MyRule) DefaultSeverity() validation.Severity { 
    return validation.SeverityWarning 
}
func (r *MyRule) Versions() []string { return nil }

func (r *MyRule) Run(ctx context.Context, docInfo *linter.DocumentInfo[*openapi.OpenAPI], config *linter.RuleConfig) []error {
    doc := docInfo.Document
    // Implement rule logic
    // Use openapi.Walk() to traverse the document
    // Return validation.Error instances for violations
    return nil
}
```

2. **Register the rule** in [`openapi/linter/linter.go`](openapi/linter/linter.go)

```go
registry.Register(&rules.MyRule{})
```

3. **Write tests** in [`openapi/linter/rules/my_rule_test.go`](openapi/linter/rules/)

```go
func TestMyRule_Success(t *testing.T) {
    t.Parallel()
    // ... test implementation
}
```

## Design Principles

1. **Generic Architecture** - The core linter is spec-agnostic (`Linter[T any]`)
2. **Type Safety** - Spec-specific rules use typed interfaces (`RuleRunner[*openapi.OpenAPI]`)
3. **Separation of Concerns** - Core engine, spec linters, and rules are separate packages
4. **Extensibility** - Easy to add new rules, rulesets, and specs
5. **Configuration Over Code** - Rule behavior controlled via YAML config
6. **Reference Resolution** - Automatic external reference resolution with proper error handling
7. **Testing** - Comprehensive test coverage with parallel execution

## Next Steps

1. Add more OpenAPI rules (e.g., security, best practices, naming conventions)
2. Create linters for other specs (Arazzo, Swagger 2.0)
3. Add auto-fix capabilities for rules that support it
4. Implement rule documentation generation in markdown/HTML formats
5. Add performance profiles and caching for large documents
