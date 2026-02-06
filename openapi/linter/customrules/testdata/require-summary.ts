// Sample custom rule that requires operations to have summaries
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
  id(): string {
    return 'custom-require-operation-summary';
  }

  category(): string {
    return 'style';
  }

  description(): string {
    return 'All operations must have a summary field for documentation.';
  }

  summary(): string {
    return 'Operations must have a summary';
  }

  run(ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
    const errors: ValidationError[] = [];

    if (!docInfo.index || !docInfo.index.operations) {
      return errors;
    }

    for (const opNode of docInfo.index.operations) {
      const op = opNode.node;
      const summary = op.getSummary ? op.getSummary() : '';

      if (!summary || summary === '') {
        const opId = op.getOperationID ? op.getOperationID() : 'unnamed';
        errors.push(createValidationError(
          config.getSeverity(this.defaultSeverity()),
          this.id(),
          `Operation "${opId}" is missing a summary`,
          op.getRootNode ? op.getRootNode() : null
        ));
      }
    }

    return errors;
  }
}

registerRule(new RequireOperationSummary());
