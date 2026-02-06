// Sample custom rule that requires operations to have descriptions
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

class RequireOperationDescription extends Rule {
  id(): string {
    return 'custom-require-operation-description';
  }

  category(): string {
    return 'style';
  }

  description(): string {
    return 'All operations must have a description field for documentation.';
  }

  summary(): string {
    return 'Operations must have a description';
  }

  run(_ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
    const errors: ValidationError[] = [];

    if (!docInfo.index || !docInfo.index.operations) {
      return errors;
    }

    for (const opNode of docInfo.index.operations) {
      const op = opNode.node;
      const description = op.getDescription ? op.getDescription() : '';

      if (!description || description === '') {
        const opId = op.getOperationID ? op.getOperationID() : 'unnamed';
        errors.push(createValidationError(
          config.getSeverity(this.defaultSeverity()),
          this.id(),
          `Operation "${opId}" is missing a description`,
          op.getRootNode ? op.getRootNode() : null
        ));
      }
    }

    return errors;
  }
}

registerRule(new RequireOperationDescription());
