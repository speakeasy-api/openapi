// Auto-generated from Spectral rule: protocol-check
// Original description: Must use approved protocols
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

class ProtocolCheck extends Rule {
  id(): string { return 'custom-protocol-check'; }
  category(): string { return 'style'; }
  description(): string { return 'Must use approved protocols'; }
  summary(): string { return 'Must use approved protocols'; }
  defaultSeverity(): 'error' | 'warning' | 'hint' { return 'error'; }

  run(_ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
    const errors: ValidationError[] = [];
    if (docInfo.index && docInfo.index.servers) {
      for (const indexNode of docInfo.index.servers) {
        const node = indexNode.node;
        // WARNING: No known getter for field 'url' — using dynamic access
if (!('url' in (node as any))) { /* field may not exist */ }
const value = (node as any)['url'];
        {
          const allowed = ['https://api.example.com', 'https://staging.example.com'];
          if (!allowed.includes(String(value))) {
            errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), 'Must use approved protocols', node.getRootNode ? node.getRootNode() : null));
          }
        }
      }
    }
    return errors;
  }
}

registerRule(new ProtocolCheck());
