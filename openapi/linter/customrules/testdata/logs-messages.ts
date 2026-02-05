// Rule that logs messages via console for testing logging
import { Rule, registerRule } from '@speakeasy-api/openapi-linter-types';
import type { Context, DocumentInfo, RuleConfig, ValidationError } from '@speakeasy-api/openapi-linter-types';

class LogsMessagesRule extends Rule {
  id(): string {
    return 'logs-messages-rule';
  }

  category(): string {
    return 'test';
  }

  description(): string {
    return 'Rule that logs messages for testing.';
  }

  summary(): string {
    return 'Logs messages';
  }

  run(_ctx: Context, docInfo: DocumentInfo, _config: RuleConfig): ValidationError[] {
    console.log('Log message from custom rule');
    console.warn('Warning message from custom rule');
    console.error('Error message from custom rule');
    console.log('Document location:', docInfo.location);
    return [];
  }
}

registerRule(new LogsMessagesRule());
