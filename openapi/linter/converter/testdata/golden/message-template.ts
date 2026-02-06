// Auto-generated from Spectral rule: message-template
// Original description: Rule with message template
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

class MessageTemplate extends Rule {
  id(): string { return 'custom-message-template'; }
  category(): string { return 'style'; }
  description(): string { return 'Rule with message template'; }
  summary(): string { return 'Rule with message template'; }
  defaultSeverity(): 'error' | 'warning' | 'hint' { return 'warning'; }

  run(_ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
    const errors: ValidationError[] = [];
    const target = docInfo.document.getInfo();
    if (target) {
      {
        const fieldValue = target.getDescription ? target.getDescription() : undefined;
        if (!fieldValue) {
          errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), 'description must be present', target.getRootNode ? target.getRootNode() : null));
        }
      }
    }
    return errors;
  }
}

registerRule(new MessageTemplate());
