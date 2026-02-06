// Auto-generated from Spectral rule: path-casing
// Original description: Path segments must use kebab-case
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

class PathCasing extends Rule {
  id(): string { return 'custom-path-casing'; }
  category(): string { return 'style'; }
  description(): string { return 'Path segments must use kebab-case'; }
  summary(): string { return 'Path segments must use kebab-case'; }
  defaultSeverity(): 'error' | 'warning' | 'hint' { return 'error'; }

  run(_ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
    const errors: ValidationError[] = [];
    if (docInfo.index && docInfo.index.inlinePathItems) {
      for (const indexNode of docInfo.index.inlinePathItems) {
        const node = indexNode.node;
        const loc = indexNode.location;
        const key = loc && loc.length > 0 ? loc[loc.length - 1].parentKey() : '';
        {
          const casingRe = /^[a-z][a-z0-9]*(-[a-z0-9]+)*$/;
          if (typeof key === 'string' && key !== '' && !casingRe.test(key)) {
            errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), 'Path segments must use kebab-case', node.getRootNode ? node.getRootNode() : null));
          }
        }
      }
    }
    return errors;
  }
}

registerRule(new PathCasing());
