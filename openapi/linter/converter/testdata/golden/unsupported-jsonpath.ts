// Auto-generated from Spectral rule: deep-path-rule
// Original description: Rule with unsupported path
import {
  Rule,
  registerRule,
  createValidationError,
} from '@speakeasy-api/openapi-linter-types';
import type {
  Context,
  DocumentInfo,
  RuleConfig,
  ValidationError,
} from '@speakeasy-api/openapi-linter-types';

class DeepPathRule extends Rule {
  id(): string { return 'custom-deep-path-rule'; }
  category(): string { return 'style'; }
  description(): string { return 'Rule with unsupported path'; }
  summary(): string { return 'Rule with unsupported path'; }
  defaultSeverity(): 'error' | 'warning' | 'hint' { return 'warning'; }

  run(_ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
    const errors: ValidationError[] = [];
    // TODO: Unsupported JSONPath: "$.x-custom.deeply.nested[*].something"
    // This rule could not be fully converted. Implement the JSONPath traversal manually.
    errors.push(createValidationError(
      config.getSeverity(this.defaultSeverity()),
      this.id(),
      'Rule not fully converted: unsupported JSONPath "$.x-custom.deeply.nested[*].something" — implement manually',
      docInfo.document.getRootNode()
    ));
    return errors;
  }
}

registerRule(new DeepPathRule());
