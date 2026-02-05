// Rule that throws an error during execution for testing error handling
import { Rule, registerRule } from '@speakeasy-api/openapi-linter-types';
import type { Context, DocumentInfo, RuleConfig, ValidationError } from '@speakeasy-api/openapi-linter-types';

class ThrowsErrorRule extends Rule {
  id(): string {
    return 'throws-error-rule';
  }

  category(): string {
    return 'test';
  }

  description(): string {
    return 'Rule that throws an error for testing.';
  }

  summary(): string {
    return 'Throws error';
  }

  run(_ctx: Context, _docInfo: DocumentInfo, _config: RuleConfig): ValidationError[] {
    throw new Error('Intentional error for testing');
  }
}

registerRule(new ThrowsErrorRule());
