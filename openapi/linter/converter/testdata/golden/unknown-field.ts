// Auto-generated from Spectral rule: extension-check
// Original description: Check x-gateway extension
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

class ExtensionCheck extends Rule {
  id(): string { return 'custom-extension-check'; }
  category(): string { return 'style'; }
  description(): string { return 'Check x-gateway extension'; }
  summary(): string { return 'Check x-gateway extension'; }
  defaultSeverity(): 'error' | 'warning' | 'hint' { return 'warning'; }

  run(_ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
    const errors: ValidationError[] = [];
    if (docInfo.index && docInfo.index.operations) {
      for (const indexNode of docInfo.index.operations) {
        const node = indexNode.node;
        {
          // WARNING: No known getter for field 'x-gateway' — using dynamic access
if (!('x-gateway' in (node as any))) { /* field may not exist */ }
const fieldValue = (node as any)['x-gateway'];
          if (!fieldValue) {
            errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), 'Check x-gateway extension', node.getRootNode ? node.getRootNode() : null));
          }
        }
      }
    }
    return errors;
  }
}

registerRule(new ExtensionCheck());
