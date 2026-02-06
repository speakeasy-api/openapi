import type { Context } from './context';
import type { RuleConfig } from './config';
import type { DocumentInfo } from './document';
import type { Severity } from './severity';
import type { ValidationError } from './validation';

/**
 * RuleRunner is the interface that custom rules must implement.
 */
export interface RuleRunner {
  /**
   * Unique identifier for this rule (e.g., "custom-require-summary").
   * Should be prefixed with "custom-" to avoid conflicts with built-in rules.
   */
  id(): string;

  /**
   * Rule category (e.g., "style", "semantic", "security", "schemas").
   */
  category(): string;

  /**
   * Detailed description of what the rule checks.
   */
  description(): string;

  /**
   * Short summary of what the rule checks.
   */
  summary(): string;

  /**
   * Optional URL to documentation for this rule.
   */
  link(): string;

  /**
   * Default severity level for this rule.
   */
  defaultSeverity(): Severity;

  /**
   * OpenAPI versions this rule applies to, or null for all versions.
   * Example: ["3.0", "3.1"] to only run on OpenAPI 3.x documents.
   */
  versions(): string[] | null;

  /**
   * Execute the rule against the document.
   *
   * @param ctx - Context for cancellation checking
   * @param docInfo - Document information including the parsed document and index
   * @param config - Rule configuration including severity overrides
   * @returns Array of validation errors found
   */
  run(ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[];
}

/**
 * Base class for implementing custom rules.
 * Provides default implementations for optional methods.
 *
 * @example
 * ```typescript
 * class RequireOperationSummary extends Rule {
 *   id(): string { return 'custom-require-operation-summary'; }
 *   category(): string { return 'style'; }
 *   description(): string { return 'All operations must have a summary.'; }
 *   summary(): string { return 'Operations must have summary'; }
 *
 *   run(ctx: Context, docInfo: DocumentInfo, config: RuleConfig) {
 *     const errors: ValidationError[] = [];
 *     for (const opNode of docInfo.index.operations) {
 *       const op = opNode.node;
 *       if (!op.getSummary()) {
 *         errors.push(createValidationError(
 *           config.getSeverity(this.defaultSeverity()),
 *           this.id(),
 *           'Operation is missing a summary',
 *           op.getRootNode()
 *         ));
 *       }
 *     }
 *     return errors;
 *   }
 * }
 *
 * registerRule(new RequireOperationSummary());
 * ```
 */
export abstract class Rule implements RuleRunner {
  abstract id(): string;
  abstract category(): string;
  abstract description(): string;
  abstract summary(): string;

  /** Default implementation returns empty string (no documentation link). */
  link(): string {
    return '';
  }

  /** Default implementation returns 'warning'. */
  defaultSeverity(): Severity {
    return 'warning';
  }

  /** Default implementation returns null (applies to all versions). */
  versions(): string[] | null {
    return null;
  }

  abstract run(ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[];
}

/**
 * Register a rule with the linter.
 * Call this at the end of your rule file to make it available.
 *
 * @param rule - The rule instance to register
 *
 * @example
 * ```typescript
 * registerRule(new MyCustomRule());
 * ```
 */
declare function registerRule(rule: RuleRunner): void;

export { registerRule };
