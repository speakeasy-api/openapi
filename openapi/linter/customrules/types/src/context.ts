/**
 * Context provides cancellation and deadline information.
 *
 * Since JavaScript has no channel equivalent to Go's Done(),
 * rules performing long operations should poll isCancelled() periodically.
 */
export interface Context {
  /**
   * Check if the context has been cancelled.
   * Rules performing long operations should poll this periodically.
   */
  isCancelled(): boolean;

  /**
   * Deadline in milliseconds since epoch, or undefined if no deadline.
   */
  deadline(): number | undefined;
}
