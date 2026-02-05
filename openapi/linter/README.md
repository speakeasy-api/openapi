# OpenAPI Linter

The OpenAPI linter validates OpenAPI specifications for style, consistency, 
and best practices beyond basic spec validation.

## Quick Start

### CLI

```bash
# Lint an OpenAPI specification
openapi spec lint api.yaml

# Output as JSON
openapi spec lint --format json api.yaml

# Disable specific rules
openapi spec lint --disable semantic-path-params api.yaml
```

### Go API

```go
import (
    "context"
    "fmt"
    "os"
    
    "github.com/speakeasy-api/openapi/linter"
    "github.com/speakeasy-api/openapi/openapi"
    openapiLinter "github.com/speakeasy-api/openapi/openapi/linter"
)

func main() {
    ctx := context.Background()
    
    // Load your OpenAPI document
    f, _ := os.Open("api.yaml")
    doc, validationErrors, _ := openapi.Unmarshal(ctx, f)
    
    // Create linter with default configuration
    config := linter.NewConfig()
    lint := openapiLinter.NewLinter(config)
    
    // Run linting
    output, _ := lint.Lint(ctx, linter.NewDocumentInfo(doc, "api.yaml"), validationErrors, nil)
    
    // Print results
    fmt.Println(output.FormatText())
}
```

## Available Rules

<!-- START LINT RULES -->

| Rule                                                                                            | Severity | Description                                                                                                                                                                                                                                                                                                                                                                                      |
| ----------------------------------------------------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| <a name="oas-schema-check"></a>`oas-schema-check`                                               | error    | Schemas must use type-appropriate constraints and have valid constraint values. For example, string types should use minLength/maxLength/pattern, numbers should use minimum/maximum/multipleOf, and constraint values must be logically valid (e.g., maxLength >= minLength).                                                                                                                   |
| <a name="oas3-example-missing"></a>`oas3-example-missing`                                       | hint     | Schemas, parameters, headers, and media types should include example values to illustrate expected data formats. Examples improve documentation quality, help developers understand how to use the API correctly, and enable better testing and validation.                                                                                                                                      |
| <a name="oas3-no-nullable"></a>`oas3-no-nullable`                                               | warning  | The nullable keyword is not supported in OpenAPI 3.1+ and should be replaced with a type array that includes null (e.g., type: [string, null]). This change aligns OpenAPI 3.1 with JSON Schema Draft 2020-12, which uses type arrays to express nullable values.                                                                                                                                |
| <a name="owasp-additional-properties-constrained"></a>`owasp-additional-properties-constrained` | hint     | Schemas with additionalProperties set to true or a schema should define maxProperties to limit object size. Without size limits, APIs are vulnerable to resource exhaustion attacks where clients send excessively large objects.                                                                                                                                                                |
| <a name="owasp-array-limit"></a>`owasp-array-limit`                                             | error    | Array schemas must specify maxItems to prevent resource exhaustion attacks. Without array size limits, malicious clients could send extremely large arrays that consume excessive memory or processing time.                                                                                                                                                                                     |
| <a name="owasp-auth-insecure-schemes"></a>`owasp-auth-insecure-schemes`                         | error    | Authentication schemes using outdated or insecure methods must be avoided or upgraded. Insecure authentication schemes like API keys in query parameters or HTTP Basic over HTTP expose credentials and create security vulnerabilities.                                                                                                                                                         |
| <a name="owasp-define-error-responses-401"></a>`owasp-define-error-responses-401`               | warning  | Operations should define a 401 Unauthorized response with a proper schema to handle authentication failures. Documenting authentication error responses helps clients implement proper error handling and understand when credentials are invalid or missing.                                                                                                                                    |
| <a name="owasp-define-error-responses-429"></a>`owasp-define-error-responses-429`               | warning  | Operations should define a 429 Too Many Requests response with a proper schema to indicate rate limiting. Rate limit responses help clients understand when they've exceeded usage thresholds and need to slow down requests.                                                                                                                                                                    |
| <a name="owasp-define-error-responses-500"></a>`owasp-define-error-responses-500`               | warning  | Operations should define a 500 Internal Server Error response with a proper schema to handle unexpected failures. Documenting server error responses helps clients distinguish between client-side and server-side problems.                                                                                                                                                                     |
| <a name="owasp-define-error-validation"></a>`owasp-define-error-validation`                     | warning  | Operations should define validation error responses (400, 422, or 4XX) to indicate request data problems. Validation error responses help clients understand when and why their request data is invalid or malformed.                                                                                                                                                                            |
| <a name="owasp-integer-format"></a>`owasp-integer-format`                                       | error    | Integer schemas must specify a format of int32 or int64 to define the expected size and range. Explicit integer formats prevent overflow vulnerabilities and ensure clients and servers agree on numeric boundaries.                                                                                                                                                                             |
| <a name="owasp-integer-limit"></a>`owasp-integer-limit`                                         | error    | Integer schemas must specify minimum and maximum values (or exclusive variants) to prevent unbounded inputs. Without numeric limits, APIs are vulnerable to overflow attacks and unexpected behavior from extreme values.                                                                                                                                                                        |
| <a name="owasp-jwt-best-practices"></a>`owasp-jwt-best-practices`                               | error    | Security schemes using OAuth2 or JWT must explicitly declare support for RFC8725 (JWT Best Current Practices) in the description. RFC8725 compliance ensures JWTs are validated properly and protected against common attacks like algorithm confusion.                                                                                                                                          |
| <a name="owasp-no-additional-properties"></a>`owasp-no-additional-properties`                   | error    | Object schemas must not allow arbitrary additional properties (set additionalProperties to false or omit it). Allowing unexpected properties can lead to mass assignment vulnerabilities where attackers inject unintended fields.                                                                                                                                                               |
| <a name="owasp-no-api-keys-in-url"></a>`owasp-no-api-keys-in-url`                               | error    | API keys must not be passed via URL parameters (query or path) as they are logged and cached. URL-based API keys appear in browser history, server logs, and proxy caches, creating security exposure.                                                                                                                                                                                           |
| <a name="owasp-no-credentials-in-url"></a>`owasp-no-credentials-in-url`                         | error    | URL parameters must not contain credentials like API keys, passwords, or secrets. Credentials in URLs are logged by servers, proxies, and browsers, creating significant security risks.                                                                                                                                                                                                         |
| <a name="owasp-no-http-basic"></a>`owasp-no-http-basic`                                         | error    | Security schemes must not use HTTP Basic authentication without additional security layers. HTTP Basic sends credentials in easily-decoded base64 encoding, making it vulnerable to interception without HTTPS.                                                                                                                                                                                  |
| <a name="owasp-no-numeric-ids"></a>`owasp-no-numeric-ids`                                       | error    | Resource identifiers must use random values like UUIDs instead of sequential numeric IDs. Sequential IDs enable enumeration attacks where attackers can guess valid IDs and access unauthorized resources.                                                                                                                                                                                       |
| <a name="owasp-protection-global-safe"></a>`owasp-protection-global-safe`                       | hint     | Safe operations (GET, HEAD) should be protected by security schemes or explicitly marked as public. Unprotected read operations may expose sensitive data to unauthorized users.                                                                                                                                                                                                                 |
| <a name="owasp-protection-global-unsafe"></a>`owasp-protection-global-unsafe`                   | error    | Unsafe operations (POST, PUT, PATCH, DELETE) must be protected by security schemes to prevent unauthorized modifications. Write operations without authentication create serious security vulnerabilities allowing data tampering.                                                                                                                                                               |
| <a name="owasp-protection-global-unsafe-strict"></a>`owasp-protection-global-unsafe-strict`     | hint     | Unsafe operations (POST, PUT, PATCH, DELETE) must be protected by non-empty security schemes without explicit opt-outs. Strict authentication requirements ensure write operations cannot bypass security even with empty security arrays.                                                                                                                                                       |
| <a name="owasp-rate-limit"></a>`owasp-rate-limit`                                               | error    | 2XX and 4XX responses must define rate limiting headers (X-RateLimit-Limit, X-RateLimit-Remaining) to prevent API overload. Rate limit headers help clients manage their usage and avoid hitting limits.                                                                                                                                                                                         |
| <a name="owasp-rate-limit-retry-after"></a>`owasp-rate-limit-retry-after`                       | error    | 429 Too Many Requests responses must include a Retry-After header indicating when clients can retry. Retry-After headers prevent thundering herd problems by telling clients exactly when to resume requests.                                                                                                                                                                                    |
| <a name="owasp-security-hosts-https-oas3"></a>`owasp-security-hosts-https-oas3`                 | error    | Server URLs must begin with https:// as the only permitted protocol. Using HTTPS is essential for protecting API traffic from interception, tampering, and eavesdropping attacks.                                                                                                                                                                                                                |
| <a name="owasp-string-limit"></a>`owasp-string-limit`                                           | error    | String schemas must specify maxLength, const, or enum to prevent unbounded data. Without string length limits, APIs are vulnerable to resource exhaustion from extremely long inputs.                                                                                                                                                                                                            |
| <a name="owasp-string-restricted"></a>`owasp-string-restricted`                                 | error    | String schemas must specify format, const, enum, or pattern to restrict content. String restrictions prevent injection attacks and ensure data conforms to expected formats.                                                                                                                                                                                                                     |
| <a name="semantic-duplicated-enum"></a>`semantic-duplicated-enum`                               | warning  | Enum arrays should not contain duplicate values. Duplicate enum values are redundant and can cause confusion or unexpected behavior in client code generation and validation.                                                                                                                                                                                                                    |
| <a name="semantic-link-operation"></a>`semantic-link-operation`                                 | error    | Link operationId must reference an existing operation in the API specification. This ensures that links point to valid operations, including those defined in external documents that are referenced in the specification.                                                                                                                                                                       |
| <a name="semantic-no-ambiguous-paths"></a>`semantic-no-ambiguous-paths`                         | error    | Path definitions must be unambiguous and distinguishable from each other to ensure correct request routing. Ambiguous paths like `/users/{id}` and `/users/{name}` can cause runtime routing conflicts since both match the same URL pattern.                                                                                                                                                    |
| <a name="semantic-no-eval-in-markdown"></a>`semantic-no-eval-in-markdown`                       | error    | Markdown descriptions must not contain eval() statements, which pose serious security risks. Including eval() in documentation could enable code injection attacks if the documentation is rendered in contexts that execute JavaScript.                                                                                                                                                         |
| <a name="semantic-no-script-tags-in-markdown"></a>`semantic-no-script-tags-in-markdown`         | error    | Markdown descriptions must not contain <script> tags, which pose serious security risks. Including script tags in documentation could enable cross-site scripting (XSS) attacks if the documentation is rendered in web contexts.                                                                                                                                                                |
| <a name="semantic-operation-id-valid-in-url"></a>`semantic-operation-id-valid-in-url`           | error    | Operation IDs must use URL-friendly characters (alphanumeric, hyphens, and underscores only). URL-safe operation IDs ensure compatibility with code generators and tooling that may use them in URLs or file paths.                                                                                                                                                                              |
| <a name="semantic-operation-operation-id"></a>`semantic-operation-operation-id`                 | warning  | Operations should define an operationId for consistent referencing across the specification and in generated code. Operation IDs enable tooling to generate meaningful function names and provide stable identifiers for API operations.                                                                                                                                                         |
| <a name="semantic-path-declarations"></a>`semantic-path-declarations`                           | error    | Path parameter declarations must not be empty - declarations like /api/{} are invalid. Empty path parameters create ambiguous routes and will cause runtime errors in most API frameworks.                                                                                                                                                                                                       |
| <a name="semantic-path-params"></a>`semantic-path-params`                                       | error    | Path template variables like {userId} must have corresponding parameter definitions with in='path', and declared path parameters must be used in the URL template. This ensures request routing works correctly and all path variables are properly documented. Parameters can be defined at PathItem level (inherited by all operations) or Operation level (can override PathItem parameters). |
| <a name="semantic-path-query"></a>`semantic-path-query`                                         | error    | Paths must not include query strings - query parameters should be defined in the parameters array instead. Including query strings in paths creates ambiguity, breaks code generation, and violates OpenAPI specification structure.                                                                                                                                                             |
| <a name="semantic-typed-enum"></a>`semantic-typed-enum`                                         | warning  | Enum values must match the specified type - for example, if type is 'string', all enum values must be strings. Type mismatches in enums cause validation failures and break code generation tools.                                                                                                                                                                                               |
| <a name="semantic-unused-component"></a>`semantic-unused-component`                             | warning  | Components that are declared but never referenced should be removed to keep the specification clean. Unused components create maintenance burden, increase specification size, and may confuse developers about which schemas are actually used.                                                                                                                                                 |
| <a name="style-component-description"></a>`style-component-description`                         | hint     | Reusable components (schemas, parameters, responses, etc.) should include descriptions to explain their purpose and usage. Clear component descriptions improve API documentation quality and help developers understand how to properly use shared definitions across the specification.                                                                                                        |
| <a name="style-contact-properties"></a>`style-contact-properties`                               | warning  | The contact object in the info section should include name, url, and email properties to provide complete contact information. Having comprehensive contact details makes it easier for API consumers to reach out for support, report issues, or ask questions about the API.                                                                                                                   |
| <a name="style-description-duplication"></a>`style-description-duplication`                     | warning  | Description and summary fields should not contain identical text within the same node. These fields serve different purposes: summaries provide brief overviews while descriptions offer detailed explanations, so duplicating content provides no additional value to API consumers.                                                                                                            |
| <a name="style-info-contact"></a>`style-info-contact`                                           | warning  | The info section should include a contact object with details for reaching the API team. Providing contact information helps API consumers get support, report issues, and connect with maintainers when needed.                                                                                                                                                                                 |
| <a name="style-info-description"></a>`style-info-description`                                   | warning  | The info section should include a description field that explains the purpose and capabilities of the API. A well-written description helps developers quickly understand what the API does and whether it meets their needs.                                                                                                                                                                    |
| <a name="style-info-license"></a>`style-info-license`                                           | hint     | The info section should include a license object that specifies the terms under which the API can be used. Clearly stating the license helps API consumers understand their rights and obligations when integrating with your API.                                                                                                                                                               |
| <a name="style-license-url"></a>`style-license-url`                                             | hint     | The license object should include a URL that points to the full license text. Providing a license URL allows API consumers to review the complete terms and conditions governing API usage.                                                                                                                                                                                                      |
| <a name="style-no-ref-siblings"></a>`style-no-ref-siblings`                                     | warning  | In OpenAPI 3.0.x, a $ref field should not have sibling properties alongside it in the same object. Either use $ref alone or move additional properties to the referenced schema definition. Note that OpenAPI 3.1+ allows $ref siblings per JSON Schema Draft 2020-12.                                                                                                                           |
| <a name="style-no-verbs-in-path"></a>`style-no-verbs-in-path`                                   | warning  | Path segments should not contain HTTP verbs like GET, POST, PUT, DELETE, or QUERY since the HTTP method already conveys the action. RESTful API design favors resource-oriented paths (e.g., `/users`) over action-oriented paths (e.g., `/getUsers`).                                                                                                                                           |
| <a name="style-oas3-api-servers"></a>`style-oas3-api-servers`                                   | warning  | OpenAPI 3.x specifications should define at least one server with a valid URL where the API can be accessed. Server definitions help API consumers understand where to send requests and support multiple environments like production, staging, and development.                                                                                                                                |
| <a name="style-oas3-host-not-example"></a>`style-oas3-host-not-example`                         | warning  | Server URLs should not point to example.com domains, which are reserved for documentation purposes. Production API specifications should reference actual server endpoints where the API is hosted.                                                                                                                                                                                              |
| <a name="style-oas3-host-trailing-slash"></a>`style-oas3-host-trailing-slash`                   | warning  | Server URLs should not end with a trailing slash to avoid ambiguity when combining with path templates. Trailing slashes can lead to double slashes in final URLs when paths are appended, potentially causing routing issues.                                                                                                                                                                   |
| <a name="style-oas3-parameter-description"></a>`style-oas3-parameter-description`               | warning  | Parameters should include descriptions that explain their purpose and expected values. Clear parameter documentation helps developers understand how to construct valid requests and what each parameter controls.                                                                                                                                                                               |
| <a name="style-openapi-tags"></a>`style-openapi-tags`                                           | warning  | The OpenAPI specification should define a non-empty tags array at the root level to organize and categorize API operations. Tags help structure API documentation and enable logical grouping of related endpoints.                                                                                                                                                                              |
| <a name="style-operation-description"></a>`style-operation-description`                         | warning  | Operations should include either a description or summary field to explain their purpose and behavior. Clear operation documentation helps developers understand what each endpoint does and how to use it effectively.                                                                                                                                                                          |
| <a name="style-operation-error-response"></a>`style-operation-error-response`                   | warning  | Operations should define at least one 4xx error response to document potential client errors. Documenting error responses helps API consumers handle failures gracefully and understand what went wrong when requests fail.                                                                                                                                                                      |
| <a name="style-operation-singular-tag"></a>`style-operation-singular-tag`                       | warning  | Operations should be associated with only a single tag to maintain clear organizational boundaries. Multiple tags can create ambiguity about where an operation belongs in the API structure and complicate documentation organization.                                                                                                                                                          |
| <a name="style-operation-success-response"></a>`style-operation-success-response`               | warning  | Operations should define at least one 2xx or 3xx response code to indicate successful execution. Success responses are essential for API consumers to understand what data they'll receive when requests complete successfully.                                                                                                                                                                  |
| <a name="style-operation-tag-defined"></a>`style-operation-tag-defined`                         | warning  | Operation tags should be declared in the global tags array at the specification root. Pre-defining tags ensures consistency, enables tag-level documentation, and helps maintain a well-organized API structure.                                                                                                                                                                                 |
| <a name="style-operation-tags"></a>`style-operation-tags`                                       | warning  | Operations should have at least one tag to enable logical grouping and organization in documentation. Tags help developers navigate the API by categorizing related operations together.                                                                                                                                                                                                         |
| <a name="style-path-trailing-slash"></a>`style-path-trailing-slash`                             | warning  | Path definitions should not end with a trailing slash to maintain consistency and avoid routing ambiguity. Trailing slashes in paths can cause mismatches with server routing rules and create duplicate endpoint definitions.                                                                                                                                                                   |
| <a name="style-paths-kebab-case"></a>`style-paths-kebab-case`                                   | warning  | Path segments should use kebab-case (lowercase with hyphens) for consistency and readability. Kebab-case paths are easier to read, follow REST conventions, and avoid case-sensitivity issues across different systems.                                                                                                                                                                          |
| <a name="style-tag-description"></a>`style-tag-description`                                     | hint     | Tags should include descriptions that explain the purpose and scope of the operations they group. Tag descriptions provide context in documentation and help developers understand the organization of API functionality.                                                                                                                                                                        |
| <a name="style-tags-alphabetical"></a>`style-tags-alphabetical`                                 | warning  | Tags should be listed in alphabetical order to improve documentation organization and navigation. Alphabetical ordering makes it easier for developers to find specific tag groups in API documentation.                                                                                                                                                                                         |

<!-- END LINT RULES -->

## Configuration

Rules can be configured via YAML configuration file or command-line flags.

```yaml
extends: speakeasy-recommended

rules:
  - id: semantic-path-params
    severity: error

  - id: validation-required-field
    match: ".*info\\.title is required.*"
    disabled: true

# Enable custom rules (see Custom Rules section below)
custom_rules:
  paths:
    - ./rules/*.ts
```

By default, the CLI loads the config from `~/.openapi/lint.yaml` unless `--config` is provided.

## Custom Rules

Write custom linting rules in TypeScript or JavaScript that run alongside the built-in rules.

### Getting Started

1. **Install the types package** in your rules directory:

```bash
npm install @speakeasy-api/openapi-linter-types
```

1. **Create a rule file** (e.g., `rules/require-summary.ts`):

```typescript
import { Rule, registerRule, createValidationError } from '@speakeasy-api/openapi-linter-types';
import type { Context, DocumentInfo, RuleConfig, ValidationError } from '@speakeasy-api/openapi-linter-types';

class RequireOperationSummary extends Rule {
  id(): string { return 'custom-require-operation-summary'; }
  category(): string { return 'style'; }
  description(): string { return 'All operations must have a summary.'; }
  summary(): string { return 'Operations must have summary'; }

  run(ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
    const errors: ValidationError[] = [];

    for (const opNode of docInfo.index.operations) {
      const op = opNode.node;
      if (!op.getSummary()) {
        errors.push(createValidationError(
          config.getSeverity(this.defaultSeverity()),
          this.id(),
          `Operation "${op.getOperationID() || 'unnamed'}" is missing a summary`,
          op.getRootNode()
        ));
      }
    }

    return errors;
  }
}

registerRule(new RequireOperationSummary());
```

1. **Configure the linter** (`lint.yaml`):

```yaml
extends: recommended

custom_rules:
  paths:
    - ./rules/*.ts

rules:
  - id: custom-require-operation-summary
    severity: error
```

### Rule Implementation

Extend the `Rule` base class and implement the required methods:

| Method                      | Required | Description                                                     |
| --------------------------- | -------- | --------------------------------------------------------------- |
| `id()`                      | Yes      | Unique identifier (prefix with `custom-`)                       |
| `category()`                | Yes      | Category for grouping (`style`, `security`, `semantic`, etc.)   |
| `description()`             | Yes      | Full description of what the rule checks                        |
| `summary()`                 | Yes      | Short summary for display                                       |
| `run(ctx, docInfo, config)` | Yes      | Main logic returning validation errors                          |
| `link()`                    | No       | URL to rule documentation                                       |
| `defaultSeverity()`         | No       | Default: `'warning'`. Options: `'error'`, `'warning'`, `'hint'` |
| `versions()`                | No       | OpenAPI versions this rule applies to (e.g., `['3.0', '3.1']`)  |

### Accessing Document Data

The `DocumentInfo` object provides access to the parsed OpenAPI document and pre-computed indices:

```typescript
run(ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
  // Access the document root
  const doc = docInfo.document;
  const info = doc.getInfo();

  // Access file location
  const location = docInfo.location;

  // Use the pre-computed index for efficient iteration
  const index = docInfo.index;

  // All operations in the document
  for (const opNode of index.operations) {
    const operation = opNode.node;
    const path = opNode.locations.path;
    const method = opNode.locations.method;
  }

  // All component schemas
  for (const schemaNode of index.componentSchemas) {
    const schema = schemaNode.node;
    const name = schemaNode.locations.name;
  }

  // All inline schemas
  for (const schemaNode of index.inlineSchemas) { ... }

  // All parameters (inline + component)
  for (const paramNode of index.parameters) { ... }

  // All request bodies
  for (const reqBodyNode of index.requestBodies) { ... }

  // All responses
  for (const responseNode of index.responses) { ... }

  // All headers
  for (const headerNode of index.headers) { ... }

  // All security schemes
  for (const secSchemeNode of index.securitySchemes) { ... }

  // ... and many more collections
}
```

### Creating Validation Errors

Use `createValidationError()` to create properly formatted errors:

```typescript
import { createValidationError } from '@speakeasy-api/openapi-linter-types';

// Basic error
errors.push(createValidationError(
  'warning',                    // severity: 'error' | 'warning' | 'hint'
  'custom-my-rule',            // rule ID
  'Description of the issue',  // message
  node.getRootNode()           // YAML node for location
));

// Using config severity (respects user overrides)
errors.push(createValidationError(
  config.getSeverity(this.defaultSeverity()),
  this.id(),
  'Description of the issue',
  node.getRootNode()
));
```

### Console Logging

The `console` global is available for debugging:

```typescript
console.log('Processing:', op.getOperationID());
console.warn('Missing recommended field');
console.error('Invalid configuration');
```

### Configuring Custom Rules

Custom rules support all standard configuration options:

```yaml
# Change severity
rules:
  - id: custom-require-operation-summary
    severity: error

# Disable a rule
rules:
  - id: custom-require-operation-summary
    disabled: true

# Filter by message pattern
rules:
  - id: custom-require-operation-summary
    match: ".*unnamed.*"
    severity: hint

# Disable entire category
categories:
  style:
    enabled: false
```

### Programmatic Usage

Enable custom rules when using the linter as a Go library:

```go
import (
    "context"

    // Import customrules package for side effects (registers the loader)
    _ "github.com/speakeasy-api/openapi/openapi/linter/customrules"

    "github.com/speakeasy-api/openapi/linter"
    "github.com/speakeasy-api/openapi/openapi"
    openapiLinter "github.com/speakeasy-api/openapi/openapi/linter"
)

func main() {
    ctx := context.Background()

    // Configure with custom rules
    config := &linter.Config{
        Extends: []string{"recommended"},
        CustomRules: &linter.CustomRulesConfig{
            Paths: []string{"./rules/*.ts"},
        },
    }

    // Create linter (custom rules are automatically loaded)
    lint, _ := openapiLinter.NewLinter(config)

    // Load and lint document
    f, _ := os.Open("api.yaml")
    doc, validationErrs, _ := openapi.Unmarshal(ctx, f)

    docInfo := linter.NewDocumentInfo(doc, "api.yaml")
    output, _ := lint.Lint(ctx, docInfo, validationErrs, nil)

    fmt.Println(output.FormatText())
}
```

### API Notes

- Field and method names use lowercase JavaScript conventions (e.g., `getSummary()`, not `GetSummary()`)
- All Go struct fields and methods are automatically exposed to JavaScript
- Arrays from the Index use JavaScript array methods (`.forEach()`, `.map()`, `.filter()`, etc.)
- Rules are transpiled with esbuild; source maps provide accurate error locations
