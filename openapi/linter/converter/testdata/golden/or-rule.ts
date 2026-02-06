// Auto-generated from Spectral rule: or-example
// Original description: Must have at least one of description or summary
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

class OrExample extends Rule {
  id(): string { return 'custom-or-example'; }
  category(): string { return 'style'; }
  description(): string { return 'Must have at least one of description or summary'; }
  summary(): string { return 'Must have at least one of description or summary'; }
  defaultSeverity(): 'error' | 'warning' | 'hint' { return 'warning'; }

  run(_ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
    const errors: ValidationError[] = [];
    if (docInfo.index && docInfo.index.operations) {
      for (const indexNode of docInfo.index.operations) {
        const node = indexNode.node;
        {
          const props = ['description', 'summary'];
          const present = props.filter(p => {
            const v = (node as any)[p];
            return v !== undefined && v !== null;
          });
          if (present.length === 0) {
            errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), 'Must have at least one of description or summary', node.getRootNode ? node.getRootNode() : null));
          }
        }
      }
    }
    return errors;
  }
}

registerRule(new OrExample());
