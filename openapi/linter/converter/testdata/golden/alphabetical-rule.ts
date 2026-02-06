// Auto-generated from Spectral rule: tags-alphabetical
// Original description: Tags must be in alphabetical order
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

class TagsAlphabetical extends Rule {
  id(): string { return 'custom-tags-alphabetical'; }
  category(): string { return 'style'; }
  description(): string { return 'Tags must be in alphabetical order'; }
  summary(): string { return 'Tags must be in alphabetical order'; }
  defaultSeverity(): 'error' | 'warning' | 'hint' { return 'warning'; }

  run(_ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
    const errors: ValidationError[] = [];
    if (docInfo.index && docInfo.index.tags) {
      for (const indexNode of docInfo.index.tags) {
        const node = indexNode.node;
        {
          const items = Array.isArray(node) ? node : [];
          for (let i = 1; i < items.length; i++) {
            const prev = items[i - 1]?.name ?? '';
            const curr = items[i]?.name ?? '';
            if (prev.localeCompare(curr) > 0) {
              errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), 'Tags must be in alphabetical order', node.getRootNode ? node.getRootNode() : null));
              break;
            }
          }
        }
      }
    }
    return errors;
  }
}

registerRule(new TagsAlphabetical());
