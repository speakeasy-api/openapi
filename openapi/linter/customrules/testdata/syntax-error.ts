// Rule with syntax error for testing error handling
class BrokenRule {
  id(): string {
    return 'broken-rule';
  }

  // Missing closing brace intentionally
  category(): string {
    return 'test'
  // syntax error: missing closing brace
