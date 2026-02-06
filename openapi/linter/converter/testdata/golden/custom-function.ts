// Auto-generated from Spectral rule: custom-fn-rule
// Original description: Rule using custom function
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

class CustomFnRule extends Rule {
  id(): string { return 'custom-custom-fn-rule'; }
  category(): string { return 'style'; }
  description(): string { return 'Rule using custom function'; }
  summary(): string { return 'Rule using custom function'; }
  defaultSeverity(): 'error' | 'warning' | 'hint' { return 'warning'; }

  run(_ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
    const errors: ValidationError[] = [];
    const target = docInfo.document.getInfo();
    if (target) {
      // TODO: Custom Spectral function "myCompanyValidator" — implement manually
      errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), 'Rule not fully converted: unsupported function "myCompanyValidator" — implement manually', docInfo.document.getRootNode()));
    }
    return errors;
  }
}

registerRule(new CustomFnRule());
