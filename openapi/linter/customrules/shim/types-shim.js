// Runtime shim for @speakeasy-api/openapi-linter-types
// This file is embedded in the Go binary and used when bundling custom rules.
// It provides the Rule base class and re-exports globals injected by the Go runtime.

// Declare the globals that are injected by the Go runtime
// These are defined in runtime.go's setupGlobals()
var registerRule = globalThis.registerRule;
var createValidationError = globalThis.createValidationError;
var createFix = globalThis.createFix;
var createValidationErrorWithFix = globalThis.createValidationErrorWithFix;

// Base Rule class - users can extend this or implement RuleRunner directly
export class Rule {
  id() { throw new Error("id() must be implemented"); }
  category() { throw new Error("category() must be implemented"); }
  description() { throw new Error("description() must be implemented"); }
  summary() { throw new Error("summary() must be implemented"); }
  link() { return ""; }
  defaultSeverity() { return "warning"; }
  versions() { return null; }
  run(ctx, docInfo, config) { throw new Error("run() must be implemented"); }
}

// Export the globals so user rules can import them
export { registerRule, createValidationError, createFix, createValidationErrorWithFix };
