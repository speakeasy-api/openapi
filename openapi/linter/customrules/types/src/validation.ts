import type { Severity } from './severity';

/**
 * ValidationError represents a linting error.
 * Created using createValidationError().
 */
export interface ValidationError {
  /** Error message */
  error(): string;

  /** Rule ID that produced this error */
  rule: string;

  /** Severity level */
  severity: Severity;

  /** Line number (1-indexed) */
  getLineNumber(): number;

  /** Column number (1-indexed) */
  getColumnNumber(): number;
}

/**
 * YamlNode represents a YAML node from the parsed document.
 * This is the node type returned by getRootNode() methods.
 */
export interface YamlNode {
  /** Node kind (scalar, mapping, sequence, etc.) */
  kind: number;

  /** Line number where this node starts (1-indexed) */
  line: number;

  /** Column number where this node starts (1-indexed) */
  column: number;

  /** Node value for scalar nodes */
  value?: string;

  /** Node tag */
  tag?: string;
}

/**
 * Creates a validation error that can be returned from a rule's run() method.
 *
 * @param severity - The error severity ('error', 'warning', or 'hint')
 * @param ruleId - The rule ID (should match rule.id())
 * @param message - The error message
 * @param node - The YAML node where the error occurred (from getRootNode())
 * @returns A validation error object
 *
 * @example
 * ```typescript
 * const error = createValidationError(
 *   'warning',
 *   this.id(),
 *   'Operation is missing a summary',
 *   operation.getRootNode()
 * );
 * ```
 */
declare function createValidationError(
  severity: Severity,
  ruleId: string,
  message: string,
  node: YamlNode | null
): ValidationError;

export { createValidationError };
