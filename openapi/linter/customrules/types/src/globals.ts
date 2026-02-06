/**
 * Global declarations for the custom rules runtime environment.
 * These are provided by the Go runtime (goja).
 *
 * When you import from '@speakeasy-api/openapi-linter-types', the global
 * `console` object is automatically typed.
 */

/**
 * Console interface for logging from custom rules.
 * Messages are routed to the configured Logger in Go.
 *
 * @example
 * ```typescript
 * console.log('Processing operation:', op.getOperationID());
 * console.warn('Missing recommended field');
 * console.error('Invalid configuration');
 * ```
 */
export interface Console {
  /** Log informational messages */
  log(...args: any[]): void;
  /** Log warning messages */
  warn(...args: any[]): void;
  /** Log error messages */
  error(...args: any[]): void;
}

// Augment the global scope to provide console type
// This is automatically included when the package is imported
declare global {
  /** Console object for logging. Provided by the goja runtime. */
  var console: Console;
}
