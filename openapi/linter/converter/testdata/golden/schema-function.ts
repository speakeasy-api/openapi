// Auto-generated from Spectral rule: schema-rule
// Original description: Schema validation rule
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

class SchemaRule extends Rule {
  id(): string { return 'custom-schema-rule'; }
  category(): string { return 'style'; }
  description(): string { return 'Schema validation rule'; }
  summary(): string { return 'Schema validation rule'; }
  defaultSeverity(): 'error' | 'warning' | 'hint' { return 'error'; }

  run(_ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
    const errors: ValidationError[] = [];
    const target = docInfo.document.getInfo();
    if (target) {
      // TODO: Function "schema" requires capabilities not available in the custom rules runtime.
      errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), 'Rule not fully converted: function "schema" — implement manually', docInfo.document.getRootNode()));
    }
    return errors;
  }
}

registerRule(new SchemaRule());
