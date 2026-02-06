// Auto-generated from Spectral rule: description-length
// Original description: Description must be between 10 and 200 characters
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

class DescriptionLength extends Rule {
  id(): string { return 'custom-description-length'; }
  category(): string { return 'style'; }
  description(): string { return 'Description must be between 10 and 200 characters'; }
  summary(): string { return 'Description must be between 10 and 200 characters'; }
  defaultSeverity(): 'error' | 'warning' | 'hint' { return 'warning'; }

  run(_ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
    const errors: ValidationError[] = [];
    if (docInfo.index && docInfo.index.operations) {
      for (const indexNode of docInfo.index.operations) {
        const node = indexNode.node;
        {
          const fieldValue = node.getDescription ? node.getDescription() : undefined;
          {
            const sLen = typeof fieldValue === 'string' || Array.isArray(fieldValue) ? fieldValue.length : (typeof fieldValue === 'number' ? fieldValue : 0);
            if (sLen < 10) {
              errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), 'Description must be between 10 and 200 characters', node.getRootNode ? node.getRootNode() : null));
            }
            if (sLen > 200) {
              errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), 'Description must be between 10 and 200 characters', node.getRootNode ? node.getRootNode() : null));
            }
          }
        }
      }
    }
    return errors;
  }
}

registerRule(new DescriptionLength());
