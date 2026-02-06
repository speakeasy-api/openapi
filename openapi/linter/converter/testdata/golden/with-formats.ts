// Auto-generated from Spectral rule: oas3-only
// Original description: OAS3 only rule
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

class Oas3Only extends Rule {
  id(): string { return 'custom-oas3-only'; }
  category(): string { return 'style'; }
  description(): string { return 'OAS3 only rule'; }
  summary(): string { return 'OAS3 only rule'; }
  defaultSeverity(): 'error' | 'warning' | 'hint' { return 'warning'; }
  versions(): string[] { return ['3.0', '3.1']; }

  run(_ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
    const errors: ValidationError[] = [];
    const target = docInfo.document.getInfo();
    if (target) {
      {
        const fieldValue = target.getDescription ? target.getDescription() : undefined;
        if (!fieldValue) {
          errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), 'OAS3 only rule', target.getRootNode ? target.getRootNode() : null));
        }
      }
    }
    return errors;
  }
}

registerRule(new Oas3Only());
