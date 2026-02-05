# @speakeasy-api/openapi-linter-types

TypeScript types for writing custom OpenAPI linter rules.

## Installation

```bash
npm install @speakeasy-api/openapi-linter-types
```

## Usage

Create a TypeScript file with your custom rule:

```typescript
import { Rule, registerRule, createValidationError } from '@speakeasy-api/openapi-linter-types';
import type { Context, DocumentInfo, RuleConfig, Severity, ValidationError } from '@speakeasy-api/openapi-linter-types';

class RequireOperationSummary extends Rule {
  id(): string { return 'custom-require-operation-summary'; }
  category(): string { return 'style'; }
  description(): string { return 'All operations must have a summary.'; }
  summary(): string { return 'Operations must have summary'; }
  defaultSeverity(): Severity { return 'warning'; }

  run(ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
    const errors: ValidationError[] = [];

    for (const opNode of docInfo.index.operations) {
      const op = opNode.node;
      if (!op.getSummary()) {
        errors.push(createValidationError(
          config.getSeverity(this.defaultSeverity()),
          this.id(),
          'Operation is missing a summary',
          op.getRootNode()
        ));
      }
    }

    return errors;
  }
}

registerRule(new RequireOperationSummary());
```

## Rule Base Class

Extend the `Rule` base class to create custom rules. The following methods must be implemented:

| Method | Description |
|--------|-------------|
| `id()` | Unique identifier for the rule (e.g., `custom-require-summary`) |
| `category()` | Category for grouping rules (e.g., `style`, `security`, `naming`) |
| `description()` | Full description of what the rule checks |
| `summary()` | Short summary for display |
| `run(ctx, docInfo, config)` | Main rule logic that returns validation errors |

Optional methods with defaults:

| Method | Default | Description |
|--------|---------|-------------|
| `link()` | `''` | URL to rule documentation |
| `defaultSeverity()` | `'warning'` | Default severity (`'error'`, `'warning'`, `'hint'`) |
| `versions()` | `null` | OpenAPI versions this rule applies to (e.g., `['3.0', '3.1']`) |

## Document Access

The `DocumentInfo` object provides access to the parsed OpenAPI document:

```typescript
// The OpenAPI document root
docInfo.document

// File location
docInfo.location

// Pre-computed index with collections for efficient iteration
docInfo.index.operations         // All operations
docInfo.index.componentSchemas   // All component schemas
docInfo.index.inlineSchemas      // All inline schemas
docInfo.index.parameters         // All parameters (inline + component)
docInfo.index.requestBodies      // All request bodies
docInfo.index.responses          // All responses
docInfo.index.headers            // All headers
docInfo.index.securitySchemes    // All security schemes
// ... and many more collections
```

## Creating Validation Errors

Use `createValidationError()` to create properly formatted errors:

```typescript
import { createValidationError } from '@speakeasy-api/openapi-linter-types';

const error = createValidationError(
  'warning',                    // severity
  'custom-my-rule',            // rule ID
  'Description of the issue',  // message
  node.getRootNode()           // YAML node for location
);
```

## Console Logging

The `console` global is available for debugging:

```typescript
console.log('Processing operation:', op.getOperationID());
console.warn('Missing recommended field');
console.error('Invalid configuration');
```

## Configuration

Configure custom rules in your `lint.yaml`:

```yaml
extends: recommended

custom_rules:
  paths:
    - ./rules/*.ts
    - ./rules/security/*.ts

rules:
  - id: custom-require-operation-summary
    severity: error
```

## API Notes

- Field and method names use lowercase JavaScript conventions (e.g., `getSummary()`, not `GetSummary()`)
- All Go struct fields and methods are automatically exposed to JavaScript
- Arrays from the Index use JavaScript array methods (`.forEach()`, `.map()`, etc.)

## License

Apache-2.0
