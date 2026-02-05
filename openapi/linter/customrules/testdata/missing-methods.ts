// Rule with missing required method (no id) for testing validation
declare function registerRule(rule: any): void;

class MissingIdRule {
  // Missing id() method - should fail validation

  category(): string {
    return 'test';
  }

  description(): string {
    return 'Rule without id method.';
  }

  summary(): string {
    return 'Missing id';
  }

  run(): any[] {
    return [];
  }
}

registerRule(new MissingIdRule());
