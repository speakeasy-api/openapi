import type { Severity } from './severity';

/**
 * RuleConfig provides configuration for a rule execution.
 */
export interface RuleConfig {
  /**
   * Get the effective severity, respecting user overrides.
   * @param defaultSeverity The rule's default severity
   * @returns The severity to use (user override or default)
   */
  getSeverity(defaultSeverity: Severity): Severity;

  /**
   * Whether the rule is enabled.
   */
  enabled(): boolean;
}
