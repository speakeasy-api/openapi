// Auto-generated from Spectral rule: xor-example
// Original description: Must have exactly one of value or externalValue
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

class XorExample extends Rule {
  id(): string { return 'custom-xor-example'; }
  category(): string { return 'style'; }
  description(): string { return 'Must have exactly one of value or externalValue'; }
  summary(): string { return 'Must have exactly one of value or externalValue'; }
  defaultSeverity(): 'error' | 'warning' | 'hint' { return 'error'; }

  run(_ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
    const errors: ValidationError[] = [];
    if (docInfo.index && docInfo.index.componentExamples) {
      for (const indexNode of docInfo.index.componentExamples) {
        const node = indexNode.node;
        {
          const props = ['value', 'externalValue'];
          const present = props.filter(p => {
            const v = (node as any)[p];
            return v !== undefined && v !== null;
          });
          if (present.length !== 1) {
            errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), 'Must have exactly one of value or externalValue', node.getRootNode ? node.getRootNode() : null));
          }
        }
      }
    }
    return errors;
  }
}

registerRule(new XorExample());
