// Auto-generated from Spectral rule: require-operation-summary
// Original description: All operations must have a summary
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

class RequireOperationSummary extends Rule {
  id(): string { return 'custom-require-operation-summary'; }
  category(): string { return 'style'; }
  description(): string { return 'All operations must have a summary'; }
  summary(): string { return 'All operations must have a summary'; }
  defaultSeverity(): 'error' | 'warning' | 'hint' { return 'warning'; }

  run(_ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
    const errors: ValidationError[] = [];
    if (docInfo.index && docInfo.index.operations) {
      for (const indexNode of docInfo.index.operations) {
        const node = indexNode.node;
        {
          const fieldValue = node.getSummary ? node.getSummary() : undefined;
          if (!fieldValue) {
            errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), 'All operations must have a summary', node.getRootNode ? node.getRootNode() : null));
          }
        }
      }
    }
    return errors;
  }
}

registerRule(new RequireOperationSummary());
