// Auto-generated from Spectral rule: multi-check
// Original description: Multiple checks on operations
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

class MultiCheck extends Rule {
  id(): string { return 'custom-multi-check'; }
  category(): string { return 'style'; }
  description(): string { return 'Multiple checks on operations'; }
  summary(): string { return 'Multiple checks on operations'; }
  defaultSeverity(): 'error' | 'warning' | 'hint' { return 'error'; }

  run(_ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
    const errors: ValidationError[] = [];
    if (docInfo.index && docInfo.index.operations) {
      for (const indexNode of docInfo.index.operations) {
        const node = indexNode.node;
        {
          const fieldValue = node.getSummary ? node.getSummary() : undefined;
          if (!fieldValue) {
            errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), 'Multiple checks on operations', node.getRootNode ? node.getRootNode() : null));
          }
        }
        {
          const fieldValue = node.getDescription ? node.getDescription() : undefined;
          if (!fieldValue) {
            errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), 'Multiple checks on operations', node.getRootNode ? node.getRootNode() : null));
          }
        }
      }
    }
    return errors;
  }
}

registerRule(new MultiCheck());
