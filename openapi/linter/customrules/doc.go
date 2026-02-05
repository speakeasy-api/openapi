// Package customrules provides support for writing OpenAPI linter rules in TypeScript/JavaScript.
//
// This package enables users to extend the OpenAPI linter with custom rules written in
// TypeScript or JavaScript, without modifying the core linter code. Rules are transpiled
// using esbuild and executed in a goja JavaScript runtime.
//
// # Usage
//
// Import this package for its side effects to enable custom rule loading:
//
//	import (
//	    _ "github.com/speakeasy-api/openapi/openapi/linter/customrules"
//	    "github.com/speakeasy-api/openapi/openapi/linter"
//	)
//
//	func main() {
//	    config := &linter.Config{
//	        Extends: []string{"recommended"},
//	        CustomRules: &linter.CustomRulesConfig{
//	            Paths: []string{"./rules/*.ts"},
//	        },
//	    }
//	    lint, err := linter.NewLinter(config)
//	    // ...
//	}
//
// # Writing Custom Rules
//
// Custom rules are written in TypeScript using the @speakeasy-api/openapi-linter-types package
// for type definitions. Rules can extend the Rule base class or implement the RuleRunner interface.
//
// Example rule (require-summary.ts):
//
//	import { Rule, registerRule, createValidationError } from '@speakeasy-api/openapi-linter-types';
//	import type { Context, DocumentInfo, RuleConfig, ValidationError } from '@speakeasy-api/openapi-linter-types';
//
//	class RequireOperationSummary extends Rule {
//	  id(): string { return 'custom-require-operation-summary'; }
//	  category(): string { return 'style'; }
//	  description(): string { return 'All operations must have a summary.'; }
//	  summary(): string { return 'Operations must have summary'; }
//
//	  run(ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
//	    const errors: ValidationError[] = [];
//
//	    for (const opNode of docInfo.index.operations) {
//	      const op = opNode.node;
//	      if (!op.getSummary()) {
//	        errors.push(createValidationError(
//	          config.getSeverity(this.defaultSeverity()),
//	          this.id(),
//	          'Operation is missing a summary',
//	          op.getRootNode()
//	        ));
//	      }
//	    }
//
//	    return errors;
//	  }
//	}
//
//	registerRule(new RequireOperationSummary());
//
// # Configuration
//
// Custom rules are configured in the linter configuration file (lint.yaml):
//
//	extends: recommended
//
//	custom_rules:
//	  paths:
//	    - ./rules/*.ts
//	    - ./rules/security/*.ts
//	  timeout: 30s  # Optional timeout per rule (default: 30s)
//
//	rules:
//	  - id: custom-require-operation-summary
//	    severity: error
//
// # API Access
//
// Custom rules have access to the full document structure through the DocumentInfo object:
//
//   - docInfo.document - The parsed OpenAPI document
//   - docInfo.location - The file path or URL of the document
//   - docInfo.index - Pre-computed index with collections for efficient iteration:
//   - index.operations - All operations in the document
//   - index.componentSchemas - All component schemas
//   - index.inlineSchemas - All inline schemas
//   - And many more collections...
//
// Field and method names are lowercased in JavaScript (e.g., getSummary(), not GetSummary())
// due to goja's UncapFieldNameMapper.
//
// # Error Handling
//
// Runtime errors in custom rules (exceptions, timeouts) are caught and returned as
// validation errors. Source maps are used to map error locations back to the original
// TypeScript source.
//
// # Thread Safety
//
// Each rule file gets its own goja runtime instance. Goja runtimes are not thread-safe,
// so rules should not share state between executions. The Loader maintains a cache of
// transpiled code but creates new runtimes for each load.
//
// # Limitations
//
//   - No npm package resolution - rules must be self-contained or use the types package
//   - No file system access from JavaScript
//   - No network access from JavaScript
//   - Memory and CPU limits should be configured via the timeout setting
package customrules
